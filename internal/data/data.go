package data

import (
	"errors"
	"strings"

	"review-service/internal/conf"
	"review-service/internal/data/query"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/glebarez/sqlite"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB)

// Data .
type Data struct {
	// TODO wrapped database client
	query *query.Query
	log   *log.Helper
	es    *elasticsearch.TypedClient
	rdb   *redis.Client
}

// NewRedisClient RedisClient的构造函数
func NewRedisClient(cfg *conf.Data) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		WriteTimeout: cfg.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  cfg.Redis.ReadTimeout.AsDuration(),
	})
}

// NewESClient ES client 的构造函数
func NewESClient(cfg *conf.Elasticsearch) (*elasticsearch.TypedClient, error) {
	c := elasticsearch.Config{
		Addresses: cfg.GetAddresses(),
	}
	// 创建客户端连接
	return elasticsearch.NewTypedClient(c)
}

// NewData .
func NewData(db *gorm.DB, esClient *elasticsearch.TypedClient, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	// 为GEN生成的query代码设置数据库连接对象
	query.SetDefault(db)
	return &Data{
		query: query.Q,
		es:    esClient,
		rdb:   rdb,
		log:   log.NewHelper(logger),
	}, cleanup, nil
}

func NewDB(cfg *conf.Data) (*gorm.DB, error) {
	switch strings.ToLower(cfg.Database.GetDriver()) {
	case "mysql":
		return gorm.Open(mysql.Open(cfg.Database.GetSource()))
	case "sqlite":
		return gorm.Open(sqlite.Open(cfg.Database.GetSource()))
	}
	return nil, errors.New("connect db failed,unsupported driver")
}
