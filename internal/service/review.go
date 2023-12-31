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
		// fmt.Printf("[service] CreateReview:err:%v\n", err)
		return &pb.CreateReviewReply{}, err
	}
	return &pb.CreateReviewReply{ReviewID: review.ReviewID}, nil
}

// GetReview 获取评价详情
func (s *ReviewService) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.GetReviewReply, error) {
	fmt.Printf("[service] GetReview req:%#v\n", req)
	review, err := s.uc.GetReview(ctx, req.GetReviewID())
	if err != nil {
		return &pb.GetReviewReply{}, nil
	}
	return &pb.GetReviewReply{
		Data: &pb.ReviewInfo{
			ReviewID:     review.ReviewID,
			UserID:       review.UserID,
			OrderID:      review.OrderID,
			Score:        review.Score,
			ServiceScore: review.ServiceScore,
			ExpressScore: review.ExpressScore,
			Content:      review.Content,
			PicInfo:      review.PicInfo,
			VideoInfo:    review.VideoInfo,
			Status:       review.Status,
		},
	}, err
}

// ListReviewByUserID 获取用户评价列表
func (s *ReviewService) ListReviewByUserID(ctx context.Context, req *pb.ListReviewByUserIDRequest) (*pb.ListReviewByUserIDReply, error) {
	fmt.Printf("[service] ListReviewByUserID req:%#v\n", req)
	var offset int = (int(req.GetPage()) - 1) * 10
	reviewInfo, err := s.uc.ListReviewByUserID(ctx, &biz.ListReviewParam{
		UserID: req.GetUserID(),
		Offset: offset,
		Size:   int(req.GetSize()),
	})
	var list []*pb.ReviewInfo
	for _, review := range reviewInfo {
		data := &pb.ReviewInfo{
			ReviewID:     review.ReviewID,
			UserID:       review.UserID,
			OrderID:      review.OrderID,
			Score:        review.Score,
			ServiceScore: review.ServiceScore,
			ExpressScore: review.ExpressScore,
			Content:      review.Content,
			PicInfo:      review.PicInfo,
			VideoInfo:    review.VideoInfo,
			Status:       review.Status,
		}
		list = append(list, data)
	}
	if err != nil {
		// fmt.Printf("[srevice] ListReviewByUserID,err:%v\n", err)
		return &pb.ListReviewByUserIDReply{}, err
	}
	return &pb.ListReviewByUserIDReply{
		List: list,
	}, nil
}

// review-B 商家端
// ReplyReview 商家回复评价
func (s *ReviewService) ReplyReview(ctx context.Context, req *pb.ReplyReviewRequest) (*pb.ReplyReviewReply, error) {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		fmt.Printf("[service] recover panic,err:%v\n", err)
	// 	}
	// }()
	fmt.Printf("[service] ReplyReview req:%#v\n", req)
	// 掉用biz层
	reply, err := s.uc.CreateReply(ctx, &biz.ReplyReviewParam{
		ReviewID:  req.ReviewID,
		StoreID:   req.StoreID,
		Content:   req.Content,
		PicInfo:   req.PicInfo,
		VideoInfo: req.VideoInfo,
	})
	if err != nil {
		fmt.Printf("[service] ReplyReview,err:%v\n", err)
		return &pb.ReplyReviewReply{}, err
	}
	return &pb.ReplyReviewReply{RelpyID: reply.ReplyID}, nil
}

// AppealReview 商家申诉评价
func (s *ReviewService) AppealReview(ctx context.Context, req *pb.AppealReviewRequest) (*pb.AppealReviewReply, error) {
	fmt.Printf("[service] AppealReview req:%#v\n", req)
	appeal, err := s.uc.AppealReview(ctx, &biz.AppealReviewParam{
		ReviewID:  req.GetReviewID(),
		StoreID:   req.GetStoreID(),
		Reason:    req.GetReason(),
		Content:   req.GetContent(),
		PicInfo:   req.GetPicInfo(),
		VideoInfo: req.GetVideoInfo(),
	})
	if err != nil {
		fmt.Printf("[service] AppealReview,err:%v\n", err)
		return &pb.AppealReviewReply{}, err
	}
	return &pb.AppealReviewReply{AppealID: appeal.AppealID}, nil
}

// review-C 运营端
// AuditReview 运营审核用户评价
func (s *ReviewService) AuditReview(ctx context.Context, req *pb.AuditReviewRequest) (*pb.AuditReviewReply, error) {
	fmt.Printf("[service] AuditReview req:%#v\n", req)
	if err := s.uc.AuditReview(ctx, &biz.AuditReviewParam{
		ReviewID:  req.GetReviewID(),
		OpUser:    req.GetOpUser(),
		OpReason:  req.GetOpReason(),
		OpRemarks: req.GetOpRemarks(),
		Status:    req.GetStatus(),
	}); err != nil {
		return &pb.AuditReviewReply{}, err
	}
	return &pb.AuditReviewReply{
		ReviewID: req.GetReviewID(),
		Status:   req.GetStatus(),
	}, nil
}

// AuditAppeal 运营审核商家申诉
func (s *ReviewService) AuditAppeal(ctx context.Context, req *pb.AuditAppealRequest) (*pb.AuditAppealReply, error) {
	fmt.Printf("[service] AuditAppeal req:%#v\n", req)
	if err := s.uc.AuditAppeal(ctx, &biz.AuditAppealParam{
		AppealID:  req.GetAppealID(),
		ReviewID:  req.GetReviewID(),
		Status:    req.GetStatus(),
		OpUser:    req.GetOpUser(),
		OpRemarks: req.GetOpRemarks(),
	}); err != nil {
		return &pb.AuditAppealReply{}, err
	}
	return &pb.AuditAppealReply{}, nil
}

func (s *ReviewService) ListReviewByStoreID(ctx context.Context, req *pb.ListReviewByStoreIDRequest) (*pb.ListReviewByStoreIDReply, error) {
	fmt.Printf("[service] ListReviewByStoreID req:%#v\n", req)
	reviewList, err := s.uc.ListReviewByStoreID(ctx, req.StoreID, int(req.Page), int(req.Size))
	if err != nil {
		return &pb.ListReviewByStoreIDReply{}, err
	}
	// fromat
	list := make([]*pb.ReviewInfo, 0, len(reviewList))
	for _, r := range reviewList {
		list = append(list, &pb.ReviewInfo{
			ReviewID:     r.ReviewID,
			UserID:       r.UserID,
			OrderID:      r.OrderID,
			Score:        r.Score,
			ServiceScore: r.ServiceScore,
			ExpressScore: r.ExpressScore,
			Content:      r.Content,
			PicInfo:      r.PicInfo,
			VideoInfo:    r.VideoInfo,
			Status:       r.Status,
		})
	}
	return &pb.ListReviewByStoreIDReply{List: list}, nil
}
