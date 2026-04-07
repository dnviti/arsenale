package queryrunner

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func openMongoTarget(ctx context.Context, target *contracts.DatabaseTarget) (*mongoTargetConn, error) {
	if target == nil {
		return nil, fmt.Errorf("target is required")
	}
	if strings.TrimSpace(target.Host) == "" {
		return nil, fmt.Errorf("target.host is required")
	}
	if target.Port <= 0 || target.Port > 65535 {
		return nil, fmt.Errorf("target.port must be between 1 and 65535")
	}
	if strings.TrimSpace(target.Username) == "" {
		return nil, fmt.Errorf("target.username is required")
	}

	database := effectiveTargetDatabase(target)
	if database == "" {
		database = "admin"
	}

	u := &url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(target.Username, target.Password),
		Host:   net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
		Path:   "/" + database,
	}
	query := u.Query()
	query.Set("appName", "arsenale-query-runner")
	query.Set("authSource", database)
	query.Set("directConnection", "true")
	u.RawQuery = query.Encode()

	clientOpts := options.Client().ApplyURI(u.String()).SetTimeout(defaultQueryTimeout)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect to target mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()
	if err := client.Database(database).RunCommand(pingCtx, bson.M{"ping": 1}).Err(); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &mongoTargetConn{
		client:   client,
		database: client.Database(database),
	}, nil
}

func (c *mongoTargetConn) Close() {
	if c == nil || c.client == nil {
		return
	}
	_ = c.client.Disconnect(context.Background())
}
