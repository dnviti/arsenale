package secretsmeta

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	ServerKey []byte
	VaultTTL  time.Duration
	ClientURL string
}
