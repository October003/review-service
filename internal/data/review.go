package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"review-service/internal/biz"
	"review-service/internal/data/model"
	"review-service/internal/data/query"
	"review-service/pkg/snowflake"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type reviewRepo struct {
	data *Data
	log  *log.Helper
}

// NewGreeterRepo .
func NewReviewRepo(data *Data, logger log.Logger) biz.ReviewRepo {
	return &reviewRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// SaveReview 保存评价到数据库中
func (r *reviewRepo) SaveReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	err := r.data.query.ReviewInfo.WithContext(ctx).Save(review)
	return review, err
}

// GetReviewByOrderID 根据订单ID查询评价
func (r *reviewRepo) GetReviewByOrderID(ctx context.Context, id int64) ([]*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.OrderID.Eq(id)).Find()
}

// GetReviewByReviewID 根据评价ID获取评价
func (r *reviewRepo) GetReviewByReviewID(ctx context.Context, id int64) (*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.ReviewID.Eq(id)).First()
}

// ListReviewByUserID 通过用户ID查询用户评价列表
func (r *reviewRepo) ListReviewByUserID(ctx context.Context, userID int64, offset, limit int) ([]*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.UserID.Eq(userID)).
		Order(r.data.query.ReviewInfo.ID.Desc()).
		Offset(offset).Limit(limit).Find()
}

// SaveReply 保存商家回复到数据库中
func (r *reviewRepo) SaveReply(ctx context.Context, reply *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error) {
	//1.数据校验
	//1.1 数据合法性校验 (已经回复的评价不允许商家再次回复)
	// 根据reviewID查询数据库，查看是否存已回复
	review, err := r.data.query.ReviewInfo.WithContext(ctx).
		Where(r.data.query.ReviewInfo.ReviewID.Eq(reply.ReviewID)).First()
	if err != nil {
		return nil, err
	}
	//判断是否已经回复
	if review.HasReply == 1 {
		return nil, errors.New("评价已回复")
	}
	//1.2 水平越权校验 (A商家只能回复自己的，不能回复B商家的评价
	// 举例子：用户A删除订单，userID + orderID，当条件去查询然后删除
	if review.StoreID != reply.StoreID {
		return nil, errors.New("水平越权")
	}
	//2. 同时更新数据库中的数据 (评价表和评价回复表要同时更新，涉及到事务操作)
	r.data.query.Transaction(func(tx *query.Query) error {
		// 评价表更新hasReply字段
		if _, err := tx.ReviewInfo.WithContext(ctx).
			Where(tx.ReviewInfo.ReviewID.Eq(review.ReviewID)).Update(tx.ReviewInfo.HasReply, 1); err != nil {
			r.log.WithContext(ctx).Errorf("UpdateReview review update fail,err:%v\n")
			return err
		}
		// 回复表插入一条数据
		if err := r.data.query.ReviewReplyInfo.WithContext(ctx).Save(reply); err != nil {
			r.log.WithContext(ctx).Errorf("SaveReply save reply fail,err:%v\n", err)
			return err
		}
		return nil
	})
	//3. 返回数据
	return reply, nil
}

// AppealReview
func (r *reviewRepo) AppealReview(ctx context.Context, param *biz.AppealReviewParam) (*model.ReviewAppealInfo, error) {
	// 1. 先查询有没有申诉
	ret, err := r.data.query.ReviewAppealInfo.WithContext(ctx).
		Where(r.data.query.ReviewAppealInfo.ReviewID.Eq(param.ReviewID),
			r.data.query.ReviewAppealInfo.StoreID.Eq(param.StoreID),
		).First()
	r.log.Debugf("AppealReview Query,ret:%v,err:%v\n", ret, err)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 其它查询错误
		return nil, err
	}
	if err == nil && ret.Status > 10 {
		return nil, errors.New("该评价已有审核过的申诉记录")
	}
	// 查询不到审核过的申诉记录
	// 1.有申诉记录但是处于待审核状态，需要更新
	// if ret != nil {
	// 	// update
	// } else {
	// 	// insert
	// }
	// 2.没有申诉记录需要创建
	appeal := &model.ReviewAppealInfo{
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Status:    10,
		Reason:    param.Reason,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}
	// 有查到申述记录 ret!=nil
	if ret != nil {
		appeal.AppealID = ret.AppealID
	} else {
		// 没有查到申诉记录，ret==nil 通过雪花算法生成AppealID
		appeal.AppealID = snowflake.GenID()
	}
	err = r.data.query.ReviewAppealInfo.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "review_id"}, // ON DUPLICATE KEY
		},
		DoUpdates: clause.Assignments(map[string]interface{}{ // UPDATE
			"status":     appeal.Status,
			"content":    appeal.Content,
			"reason":     appeal.Reason,
			"pic_info":   appeal.PicInfo,
			"video_info": appeal.VideoInfo,
		}),
	},
	).Create(appeal) // INSERT
	r.log.Debugf("AppealReview,err:%v\n", err)
	return appeal, err
}

// AuditReview 审核用户评价 (运营对用户的评价进行审核)
func (r *reviewRepo) AuditReview(ctx context.Context, param *biz.AuditReviewParam) error {
	_, err := r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.ReviewID.Eq(param.ReviewID)).
		Updates(map[string]interface{}{
			"status":     param.Status,
			"op_user":    param.OpUser,
			"op_reason":  param.OpReason,
			"op_remarks": param.OpRemarks,
		})
	return err
}

// AuditAppeal 审核商家申诉 (运营对商家的申诉进行审核 ,审核通过会隐藏该评价)
func (r *reviewRepo) AuditAppeal(ctx context.Context, param *biz.AuditAppealParam) error {
	err := r.data.query.Transaction(func(tx *query.Query) error {
		// 申诉表
		if _, err := tx.ReviewAppealInfo.WithContext(ctx).Where(tx.ReviewAppealInfo.AppealID.Eq(param.AppealID)).
			Updates(map[string]interface{}{
				"status":  param.Status,
				"op_user": param.OpUser,
			}); err != nil {
			return err
		}
		// 评价表
		// 申诉通过需要隐藏评价
		if param.Status == 20 {
			if _, err := tx.ReviewInfo.WithContext(ctx).Where(tx.ReviewInfo.ReviewID.Eq(param.ReviewID)).
				Update(tx.ReviewInfo.Status, 40); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (r *reviewRepo) ListReviewByStoreID(ctx context.Context, storeID int64, offset, limit int) ([]*biz.MyReviewInfo, error) {
	// return r.getData1(ctx,storeID,offset,limit) // 第一版 直接查es
	return r.getData2(ctx, storeID, offset, limit) // 第二版 增加缓存和singleflight
}

func (r *reviewRepo) getData1(ctx context.Context, storeID int64, offset, limit int) ([]*biz.MyReviewInfo, error) {
	// 去Elasticsearch 查询评价
	resp, err := r.data.es.Search().Index("review").From(offset).Size(limit).
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{
					{
						Term: map[string]types.TermQuery{
							"store_id": {Value: storeID},
						},
					},
				},
			},
		}).Do(ctx)
	fmt.Printf("--> es search: %v %v\n", resp, err)
	if err != nil {
		return nil, err
	}
	fmt.Printf("es result total:%v\n", resp.Hits.Total.Value)
	// 反序列化数据
	list := make([]*biz.MyReviewInfo, 0, resp.Hits.Total.Value)
	for _, hit := range resp.Hits.Hits {
		tmp := &biz.MyReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.Errorf("json.Unmarshal(hit.Source_,tmp) failed,err:%v\n", err)
			continue
		}
		list = append(list, tmp)
	}
	return nil, nil

}

// getData2 升级版 带缓存版本的查询函数
func (r *reviewRepo) getData2(ctx context.Context, storeID int64, offset, limit int) ([]*biz.MyReviewInfo, error) {
	// 取数据
	// 1.先查询Redis缓存
	// 2.缓存没有则查询 ES
	// 3.通过singleflight 合并短时间内大量的并发请求
	key := fmt.Sprintf("review:%d:%d:%d", storeID, offset, limit)
	b, err := r.getDataBySingleflight(ctx, key)
	if err != nil {
		return nil, err
	}
	hm := new(types.HitsMetadata)
	if err := json.Unmarshal(b, hm); err != nil {
		return nil, err
	}
	// 反序列化数据
	// resp.Hits.Hits[0].Source_(json.RawMessage) --> biz.MyReviewInfo
	list := make([]*biz.MyReviewInfo, 0, hm.Total.Value)
	for _, hit := range hm.Hits {
		tmp := &biz.MyReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.Errorf("json.Unmarshal(hit.Source_,tmp) failed,err:%v\n", err)
			continue
		}
		list = append(list, tmp)
	}
	return list, nil
}

var g singleflight.Group

// key review:76089:1:10 --> "[{},{},{}]"
// josn.Unmarshal([]byte)
// getDataBySingleflight
func (r *reviewRepo) getDataBySingleflight(ctx context.Context, key string) ([]byte, error) {
	v, err, shared := g.Do(key, func() (interface{}, error) {
		// 查缓存
		data, err := r.getDataFromCache(ctx, key)
		r.log.Debugf("r.getDataFromCache(ctx,key) data:%s,err:%v\n", data, err)
		if err == nil {
			return data, nil
		}
		// 只有返回缓存中没有这个key的错误时 才查询es
		if errors.Is(err, redis.Nil) {
			// 缓存中没有这个key，说明缓存失效了，要查询es
			data, err := r.getDataFromES(ctx, key)
			if err == nil {
				// 设置缓存
				return data, r.setCache(ctx, key, data)
			}
		}
		// 查询缓存失败了，直接返回错误，不继续向下传导压力
		return nil, err
	})
	r.log.Debugf("singleflight result: v:%v err:%v shared:%v\n", v, err, shared)
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

// getDataFromCache 读缓存
func (r *reviewRepo) getDataFromCache(ctx context.Context, key string) ([]byte, error) {
	r.log.Debugf("getDataFromCache key:%v\n", key)
	return r.data.rdb.Get(ctx, key).Bytes()
}

// setCache 设置缓存
func (r *reviewRepo) setCache(ctx context.Context, key string, data []byte) error {
	r.log.Debugf("setCahce key:%v\t,data:%s\n", key, data)
	return r.data.rdb.Set(ctx, key, data, time.Second*60).Err()
}

// getDataFromES 从es中查询
func (r *reviewRepo) getDataFromES(ctx context.Context, key string) ([]byte, error) {
	values := strings.Split(key, ":")
	if len(values) < 4 {
		return nil, errors.New("invalid key")
	}
	index, storeID, offsetStr, limitStr := values[0], values[1], values[2], values[3]
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return nil, err
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return nil, err
	}
	resp, err := r.data.es.Search().Index(index).From(offset).Size(limit).
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{
					{
						Term: map[string]types.TermQuery{
							"store_id": {Value: storeID},
						},
					},
				},
			},
		}).Do(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp.Hits)
}
