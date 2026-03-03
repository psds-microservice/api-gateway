package controller

import (
	"context"
	"encoding/json"
	"time"

	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"github.com/redis/go-redis/v9"
)

// RedisRepository implements StreamStore and ClientStore using Redis.
type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

const (
	streamKeyPrefix  = "psds:stream:"
	statsKeyPrefix   = "psds:stats:"
	clientKeyPrefix  = "psds:client:"
	activeStreamsKey = "psds:active_streams"
)

// StreamStore Implementation

func (r *RedisRepository) SaveStream(ctx context.Context, streamID string, stream *pb.ActiveStream) {
	data, _ := json.Marshal(stream)
	r.client.Set(ctx, streamKeyPrefix+streamID, data, 24*time.Hour)
	r.client.SAdd(ctx, activeStreamsKey, streamID)

	// Initialize stats if not exist
	key := statsKeyPrefix + streamID
	if r.client.Exists(ctx, key).Val() == 0 {
		stats := &pb.StreamStats{
			StreamId:    streamID,
			ClientId:    stream.ClientId,
			StartTime:   time.Now().Unix(),
			Codec:       "H.264",
			Width:       1920,
			Height:      1080,
			IsRecording: stream.IsRecording,
			IsStreaming: stream.IsStreaming,
		}
		statsData, _ := json.Marshal(stats)
		r.client.Set(ctx, key, statsData, 24*time.Hour)
	}
}

func (r *RedisRepository) GetStream(ctx context.Context, streamID string) *pb.ActiveStream {
	data, err := r.client.Get(ctx, streamKeyPrefix+streamID).Bytes()
	if err != nil {
		return nil
	}
	var stream pb.ActiveStream
	if err := json.Unmarshal(data, &stream); err != nil {
		return nil
	}
	return &stream
}

func (r *RedisRepository) GetStats(ctx context.Context, streamID string) *pb.StreamStats {
	data, err := r.client.Get(ctx, statsKeyPrefix+streamID).Bytes()
	if err != nil {
		return nil
	}
	var stats pb.StreamStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil
	}
	return &stats
}

func (r *RedisRepository) UpdateStats(ctx context.Context, streamID string, frame *pb.VideoFrame) *pb.StreamStats {
	stats := r.GetStats(ctx, streamID)
	if stats == nil {
		return nil
	}

	stats.FramesReceived++
	if frame != nil {
		stats.BytesReceived += int64(len(frame.FrameData))
		if frame.Width > 0 {
			stats.Width = frame.Width
		}
		if frame.Height > 0 {
			stats.Height = frame.Height
		}
	}
	now := time.Now().Unix()
	stats.Duration = now - stats.StartTime
	if stats.Duration > 0 {
		stats.AverageFps = float32(float64(stats.FramesReceived) / float64(stats.Duration))
	}

	statsData, _ := json.Marshal(stats)
	r.client.Set(ctx, statsKeyPrefix+streamID, statsData, 24*time.Hour)
	return stats
}

func (r *RedisRepository) RemoveStream(ctx context.Context, streamID string) {
	r.client.Del(ctx, streamKeyPrefix+streamID)
	r.client.Del(ctx, statsKeyPrefix+streamID)
	r.client.SRem(ctx, activeStreamsKey, streamID)
}

func (r *RedisRepository) GetAllActiveStreams(ctx context.Context) []*pb.ActiveStream {
	ids := r.client.SMembers(ctx, activeStreamsKey).Val()
	streams := make([]*pb.ActiveStream, 0, len(ids))
	for _, id := range ids {
		if s := r.GetStream(ctx, id); s != nil {
			if s.IsRecording || s.IsStreaming {
				streams = append(streams, s)
			}
		}
	}
	return streams
}

func (r *RedisRepository) GetAllStreams(ctx context.Context) []*pb.ActiveStream {
	ids := r.client.SMembers(ctx, activeStreamsKey).Val()
	streams := make([]*pb.ActiveStream, 0, len(ids))
	for _, id := range ids {
		if s := r.GetStream(ctx, id); s != nil {
			streams = append(streams, s)
		}
	}
	return streams
}

func (r *RedisRepository) GetAllStats(ctx context.Context) []*pb.StreamStats {
	ids := r.client.SMembers(ctx, activeStreamsKey).Val()
	statsList := make([]*pb.StreamStats, 0, len(ids))
	for _, id := range ids {
		if s := r.GetStats(ctx, id); s != nil {
			statsList = append(statsList, s)
		}
	}
	return statsList
}

// ClientStore Implementation

func (r *RedisRepository) SaveClient(ctx context.Context, client *pb.ClientInfo) {
	if client == nil {
		return
	}
	if client.Stats == nil {
		client.Stats = &pb.ClientInfo_ClientStats{}
	}
	client.Stats.LastActivity = time.Now().Unix()
	data, _ := json.Marshal(client)
	r.client.Set(ctx, clientKeyPrefix+client.ClientId, data, 24*time.Hour)
}

func (r *RedisRepository) GetClient(ctx context.Context, clientID string) *pb.ClientInfo {
	data, err := r.client.Get(ctx, clientKeyPrefix+clientID).Bytes()
	if err != nil {
		return nil
	}
	var client pb.ClientInfo
	if err := json.Unmarshal(data, &client); err != nil {
		return nil
	}
	return &client
}

func (r *RedisRepository) RemoveClient(ctx context.Context, clientID string) {
	r.client.Del(ctx, clientKeyPrefix+clientID)
}

func (r *RedisRepository) GetAllClients(ctx context.Context) []*pb.ClientInfo {
	// Simple pattern scan for education purposes; in production, use a Set of IDs.
	keys := r.client.Keys(ctx, clientKeyPrefix+"*").Val()
	clients := make([]*pb.ClientInfo, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err == nil {
			var client pb.ClientInfo
			if err := json.Unmarshal(data, &client); err == nil {
				clients = append(clients, &client)
			}
		}
	}
	return clients
}
