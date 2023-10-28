package biz

import (
	"context"
	"errors"
	"fmt"
	"review-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
	GetReviewByOrderID(context.Context, int64) ([]*model.ReviewInfo, error)
}

type ReviewUsecase struct {
	repo ReviewRepo
	log  *log.Helper
}

func NewReviewUsecase(repo ReviewRepo, logger log.Logger) *ReviewUsecase {
	return &ReviewUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// 实现业务逻辑的地方
// service层调用该方法
func (uc *ReviewUsecase) CreateReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] CreateReview, req:%v", review)
	// 1.数据校验
	// 1.1 参数基础校验: 正常来说不应该放在这一层，在上一层或者框架层拦住
	// 1.2 参数业务校验: 带业务逻辑的参数校验，比如已经评价过的订单不能再创建评价
	reviews, err := uc.repo.GetReviewByOrderID(ctx, review.OrderID)
	if err != nil {
		return nil, errors.New("查询数据库失败")
	}
	if len(reviews) > 0 {
		return nil,fmt.Errorf("订单%d已评价",review.OrderID)
	}
	// 2.生成reviewID (雪花算法)
	// 这里可以使用雪花算法自己生成

	// 3.查询订单和商品快照信息

	// 4.拼装数据入库

	return uc.repo.SaveReview(ctx, review)
}
