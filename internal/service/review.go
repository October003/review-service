package service

import (
	"context"
	"fmt"

	pb "review-service/api/review/v1"

	"review-service/internal/biz"
	"review-service/internal/data/model"
)

type ReviewService struct {
	pb.UnimplementedReviewServer
	uc *biz.ReviewUsecase
}

func NewReviewService(uc *biz.ReviewUsecase) *ReviewService {
	return &ReviewService{uc: uc}
}

// CreateReview 创建评价
func (s *ReviewService) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.CreateReviewReply, error) {
	fmt.Printf("[service] CreateReview req:%#v\n", req)
	// 判是否为匿名评价
	var anonymous int32
	if req.Anonymous {
		anonymous = 1
	}
	review, err := s.uc.CreateReview(ctx, &model.ReviewInfo{
		UserID:       req.UserID,
		OrderID:      req.OrderID,
		Score:        req.Score,
		ServiceScore: req.ServiceScore,
		ExpressScore: req.ExpressScore,
		Content:      req.Content,
		PicInfo:      req.PicInfo,
		VideoInfo:    req.VideoInfo,
		Anonymous:    anonymous,
		Status:       0,
	})
	if err != nil {
		return &pb.CreateReviewReply{}, nil
	}
	return &pb.CreateReviewReply{ReviewID: review.ReviewID}, nil
}

// GetReview 获取评价详情
func (s *ReviewService) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.GetReviewReply, error) {
	return &pb.GetReviewReply{}, nil
}

// ListReviewByUserID 获取用户评价列表
func (s *ReviewService) ListReviewByUserID(ctx context.Context,req *pb.ListReviewByUserIDRequest) (*pb.ListReviewByUserIDReply,error){

	return &pb.ListReviewByUserIDReply{},nil
}

// ReplyReview 商家回复评价
func (s *ReviewService) ReplyReview(ctx context.Context, req *pb.ReplyReviewRequest) (*pb.ReplyReviewReply, error) {
	fmt.Printf("[service] ReplyReview req:%#v\n", req)
	// 掉用biz层
	reply, err := s.uc.ReviewReply(ctx, &biz.ReplyParam{
		ReviewID:  req.ReviewID,
		StoreID:   req.StoreID,
		Content:   req.Content,
		PicInfo:   req.PicInfo,
		VideoInfo: req.VideoInfo,
	})
	if err != nil {
		return &pb.ReplyReviewReply{}, nil
	}
	return &pb.ReplyReviewReply{RelpyID: *reply.ReplyID}, nil
}
// AppealReview 商家申诉评价
func (s *ReviewService) AppealReview(ctx context.Context, req *pb.AppealReviewRequest) (*pb.AppealReviewReply, error) {
	return &pb.AppealReviewReply{}, nil
}

// AuditReview 运营审核用户评价
func (s *ReviewService) AuditReview(ctx context.Context, req *pb.AuditReviewRequest) (*pb.AuditReviewReply, error) {
	return &pb.AuditReviewReply{}, nil
}

// AuditAppeal 运营审核商家申诉
func(s *ReviewService) AuditAppeal(ctx context.Context,req *pb.AuditAppealRequest) (*pb.AuditAppealReply,error){
	return &pb.AuditAppealReply{},nil
}