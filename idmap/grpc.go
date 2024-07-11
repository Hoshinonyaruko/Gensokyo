package idmap

import (
	"context"

	"github.com/hoshinonyaruko/gensokyo/proto"
)

type Server struct {
	proto.UnimplementedIDMapServiceServer
}

func (s *Server) StoreIDV2(ctx context.Context, req *proto.StoreIDRequest) (*proto.StoreIDResponse, error) {
	newRow, err := StoreIDv2(req.IdOrRow)
	if err != nil {
		return nil, err
	}
	return &proto.StoreIDResponse{Row: newRow}, nil
}

func (s *Server) RetrieveRowByIDV2(ctx context.Context, req *proto.RetrieveRowByIDRequest) (*proto.RetrieveRowByIDResponse, error) {
	id, err := RetrieveRowByIDv2(req.IdOrRow)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveRowByIDResponse{Id: id}, nil
}

func (s *Server) WriteConfigV2(ctx context.Context, req *proto.WriteConfigRequest) (*proto.WriteConfigResponse, error) {
	err := WriteConfigv2(req.Section, req.Subtype, req.Value)
	if err != nil {
		return nil, err
	}
	return &proto.WriteConfigResponse{Status: "success"}, nil
}

func (s *Server) ReadConfigV2(ctx context.Context, req *proto.ReadConfigRequest) (*proto.ReadConfigResponse, error) {
	value, err := ReadConfigv2(req.Section, req.Subtype)
	if err != nil {
		return nil, err
	}
	return &proto.ReadConfigResponse{Value: value}, nil
}

func (s *Server) UpdateVirtualValueV2(ctx context.Context, req *proto.UpdateVirtualValueRequest) (*proto.UpdateVirtualValueResponse, error) {
	err := UpdateVirtualValuev2(req.OldVirtualValue, req.NewVirtualValue)
	if err != nil {
		return nil, err
	}
	return &proto.UpdateVirtualValueResponse{Status: "success"}, nil
}

func (s *Server) RetrieveRealValueV2(ctx context.Context, req *proto.RetrieveRealValueRequest) (*proto.RetrieveRealValueResponse, error) {
	virtual, real, err := RetrieveRealValuev2(req.VirtualValue)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveRealValueResponse{Virtual: virtual, Real: real}, nil
}

func (s *Server) RetrieveRealValueV2Pro(ctx context.Context, req *proto.RetrieveRealValueRequestPro) (*proto.RetrieveRealValueResponsePro, error) {
	virtual, real, err := RetrieveRealValuePro(req.VirtualValue, req.VirtualValueSub)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveRealValueResponsePro{Virtual: virtual, Real: real}, nil
}

func (s *Server) RetrieveVirtualValueV2(ctx context.Context, req *proto.RetrieveVirtualValueRequest) (*proto.RetrieveVirtualValueResponse, error) {
	real, virtual, err := RetrieveVirtualValuev2(req.RealValue)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveVirtualValueResponse{Real: real, Virtual: virtual}, nil
}

func (s *Server) StoreIDV2Pro(ctx context.Context, req *proto.StoreIDProRequest) (*proto.StoreIDProResponse, error) {
	row, subRow, err := StoreIDv2Pro(req.IdOrRow, req.Subid)
	if err != nil {
		return nil, err
	}
	return &proto.StoreIDProResponse{Row: row, SubRow: subRow}, nil
}

func (s *Server) RetrieveRowByIDV2Pro(ctx context.Context, req *proto.RetrieveRowByIDProRequest) (*proto.RetrieveRowByIDProResponse, error) {
	id, subid, err := RetrieveRowByIDv2Pro(req.IdOrRow, req.Subid)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveRowByIDProResponse{Id: id, Subid: subid}, nil
}

func (s *Server) RetrieveVirtualValueV2Pro(ctx context.Context, req *proto.RetrieveVirtualValueProRequest) (*proto.RetrieveVirtualValueProResponse, error) {
	firstValue, secondValue, err := RetrieveVirtualValuev2Pro(req.IdOrRow, req.Subid)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveVirtualValueProResponse{FirstValue: firstValue, SecondValue: secondValue}, nil
}

func (s *Server) UpdateVirtualValueV2Pro(ctx context.Context, req *proto.UpdateVirtualValueProRequest) (*proto.UpdateVirtualValueProResponse, error) {
	err := UpdateVirtualValuev2Pro(req.OldVirtualValue_1, req.NewVirtualValue_1, req.OldVirtualValue_2, req.NewVirtualValue_2)
	if err != nil {
		return nil, err
	}
	return &proto.UpdateVirtualValueProResponse{Message: "Virtual values updated successfully"}, nil
}

func (s *Server) SimplifiedStoreIDV2(ctx context.Context, req *proto.SimplifiedStoreIDRequest) (*proto.SimplifiedStoreIDResponse, error) {
	row, err := SimplifiedStoreIDv2(req.IdOrRow)
	if err != nil {
		return nil, err
	}
	return &proto.SimplifiedStoreIDResponse{Row: row}, nil
}

func (s *Server) FindSubKeysByIdPro(ctx context.Context, req *proto.FindSubKeysRequest) (*proto.FindSubKeysResponse, error) {
	keys, err := FindSubKeysByIdPro(req.Id)
	if err != nil {
		return nil, err
	}
	return &proto.FindSubKeysResponse{Keys: keys}, nil
}

func (s *Server) DeleteConfigV2(ctx context.Context, req *proto.DeleteConfigRequest) (*proto.DeleteConfigResponse, error) {
	err := DeleteConfigv2(req.Section, req.Subtype)
	if err != nil {
		return nil, err
	}
	return &proto.DeleteConfigResponse{Status: "success"}, nil
}

func (s *Server) StoreCacheV2(ctx context.Context, req *proto.StoreCacheRequest) (*proto.StoreCacheResponse, error) {
	row, err := StoreCachev2(req.IdOrRow)
	if err != nil {
		return nil, err
	}
	return &proto.StoreCacheResponse{Row: row}, nil
}

func (s *Server) RetrieveRowByCacheV2(ctx context.Context, req *proto.RetrieveRowByCacheRequest) (*proto.RetrieveRowByCacheResponse, error) {
	id, err := RetrieveRowByCachev2(req.IdOrRow)
	if err != nil {
		return nil, err
	}
	return &proto.RetrieveRowByCacheResponse{Id: id}, nil
}
