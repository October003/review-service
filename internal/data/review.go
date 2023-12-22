package data

import (
	"context"
	"errors"

	"review-service/internal/biz"
	"review-service/internal/data/model"
	"review-service/internal/data/query"
	"review-service/pkg/snowflake"

	"github.com/go-kratos/kratos/v2/log"
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
