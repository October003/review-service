package data

import (
	"context"
	"errors"

	"review-service/internal/biz"
	"review-service/internal/data/model"
	"review-service/internal/data/query"

	"github.com/go-kratos/kratos/v2/log"
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

// SaveReply 保存商家回复到数据库中
func (r *reviewRepo) SaveReply(ctx context.Context, reply *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error) {
	//1.数据校验
	//1.1 数据合法性校验 (已经回复的评价不允许再回复)
	// 根据reviewID查询这条评价
	review, err := r.data.query.ReviewInfo.WithContext(ctx).
		Where(r.data.query.ReviewInfo.ReviewID.Eq(*reply.ReviewID)).First()
	if err != nil {
		return nil, nil
	}
	//判断是否已经回复
	if review.HasReply == 1 {
		return nil, errors.New("评价已回复")
	}
	//1.2 水平越权校验 (A商家不能回复B商家的评价)
	if review.StoreID != *reply.StoreID {
		return nil, errors.New("水平越权")
	}
	//2. 同时更新评价表和评价回复表中的数据
	r.data.query.Transaction(func(tx *query.Query) error {
		if _, err := tx.ReviewInfo.WithContext(ctx).
			Where(tx.ReviewInfo.ReviewID.Eq(review.ReviewID)).Update(tx.ReviewInfo.HasReply, 1); err != nil {
			r.log.Errorf("UpdateReview review update fail,err:%v\n")
			return err
		}
		if err := r.data.query.ReviewReplyInfo.WithContext(ctx).Save(reply); err != nil {
			r.log.Error("SaveReply save reply fail,err:%v\n", err)
			return err
		}
		return nil
	})
	//3. 返回数据
	return reply, nil
}

