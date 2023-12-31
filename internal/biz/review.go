package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "review-service/api/review/v1"
	"review-service/internal/data/model"
	"review-service/pkg/snowflake"

	"github.com/go-kratos/kratos/v2/log"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
	GetReviewByOrderID(context.Context, int64) ([]*model.ReviewInfo, error)
	GetReviewByReviewID(context.Context, int64) (*model.ReviewInfo, error)
	ListReviewByUserID(context.Context, int64, int, int) ([]*model.ReviewInfo, error)
	SaveReply(context.Context, *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error)
	AppealReview(context.Context, *AppealReviewParam) (*model.ReviewAppealInfo, error)
	AuditReview(context.Context, *AuditReviewParam) error
	AuditAppeal(context.Context, *AuditAppealParam) error
	ListReviewByStoreID(context.Context,storeID int64,offset , limit int ) ([]*MyReviewInfo,error)
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
		return nil, v1.ErrorDbFailed("查询数据库失败")
	}
	if len(reviews) > 0 {
		fmt.Printf("len(reviews):%d", len(reviews))
		return nil, v1.ErrorOrderReviewed("订单%d已评价", review.OrderID)
	}
	// 2.生成reviewID (雪花算法)
	// 这里可以使用雪花算法自己生成
	review.ReviewID = snowflake.GenID()
	// 3.查询订单和商品快照信息
	// 实际业务场景下就需要查询订单服务和商家服务(比如说通过RPC调用订单服务和商家服务)
	// 4.拼装数据入库
	fmt.Printf("[biz] CreateReview,review:%#v\n", review)
	return uc.repo.SaveReview(ctx, review)
}

// GetReview
func (uc *ReviewUsecase) GetReview(ctx context.Context, reviewId int64) (*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] GetReview,req:%#v\n", reviewId)
	return uc.repo.GetReviewByReviewID(ctx, reviewId)
}

// ListReviewByUserID 通过用户id获取评价列表
func (uc *ReviewUsecase) ListReviewByUserID(ctx context.Context, param *ListReviewParam) ([]*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] ListReviewByUserID,param:%#v\n", param)
	return uc.repo.ListReviewByUserID(ctx, param.UserID, param.Offset, param.Size)
}

func (uc *ReviewUsecase) CreateReply(ctx context.Context, param *ReplyReviewParam) (*model.ReviewReplyInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] ReviewReply,param:%v", param)
	reply := &model.ReviewReplyInfo{
		ReplyID:   snowflake.GenID(),
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}
	return uc.repo.SaveReply(ctx, reply)
}

// AppealReview
func (uc *ReviewUsecase) AppealReview(ctx context.Context, param *AppealReviewParam) (*model.ReviewAppealInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] AppealReview,param:%#v\n", param)
	return uc.repo.AppealReview(ctx, param)
}

// AuditReview
func (uc *ReviewUsecase) AuditReview(ctx context.Context, param *AuditReviewParam) error {
	uc.log.WithContext(ctx).Debugf("[biz] AuditReview,param:%#v\n", param)
	return uc.repo.AuditReview(ctx, param)
}

// AuditAppeal
func (uc *ReviewUsecase) AuditAppeal(ctx context.Context, param *AuditAppealParam) error {
	uc.log.WithContext(ctx).Debugf("[biz] AuditAppeal,param:%#v\n", param)
	return uc.repo.AuditAppeal(ctx, param)
}

// ListReviewByStoreID 根据storeID分页查询评价
func (uc *ReviewUsecase) ListReviewByStoreID(ctx context.Context, storeID int64, page, size int) ([]*MyReviewInfo, error) {
	if page < 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size
	limit := size
	uc.log.WithContext(ctx).Debugf("[biz] ListReviewByStoreID storeID:%v\n", storeID)
	return uc.repo.ListReviewByStoreID(ctx, storeID, offset, limit)
}

type MyReviewInfo struct {
	*model.ReviewInfo
	CreateAt     MyTime `json:"create_at"` // 创建时间
	UpdateAt     MyTime `json:"update_at"` // 创建时间
	Anonymous    int32  `json:"anonymous,string"`
	Score        int32  `json:"score,string"`
	ServiceScore int32  `json:"service_score,string"`
	ExpressScore int32  `json:"express_score,string"`
	HasMedia     int32  `json:"has_media,string"`
	Status       int32  `json:"status,string"`
	IsDefault    int32  `json:"is_default,string"`
	HasReply     int32  `json:"has_reply,string"`
	ID           int64  `json:"id,string"`
	Version      int32  `json:"version,string"`
	ReviewID     int64  `json:"review_id,string"`
	OrderID      int64  `json:"order_id,string"`
	SkuID        int64  `json:"sku_id,string"`
	SpuID        int64  `json:"spu_id,string"`
	StoreID      int64  `json:"store_id,string"`
	UserID       int64  `json:"user_id,string"`
}

type MyTime time.Time

// UnmarshalJSON json.Unmarshal 的时候会自动调用这个方法
func (t *MyTime) UnmarshalJSON(data []byte) error {
	// data = "\"2023-12-17 20:03:54\""
	s := strings.Trim(string(data), `"`)
	tmp, err := time.Parse(time.DateTime, s)
	if err != nil {
		return err
	}
	*t = MyTime(tmp)
	return nil
}

