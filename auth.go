// Copyright 2017 Ross Peoples
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mydis

import (
	"errors"

	"github.com/coreos/etcd/auth/authpb"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/deejross/mydis/pb"
	"golang.org/x/net/context"
)

// AuthEnable enabled authentication.
func (s *Server) AuthEnable(ctx context.Context, req *pb.AuthEnableRequest) (*pb.AuthEnableResponse, error) {
	resp, err := s.cache.Server.AuthEnable(ctx, &etcdserverpb.AuthEnableRequest{})
	if err != nil {
		return &pb.AuthEnableResponse{}, err
	}
	return &pb.AuthEnableResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// AuthDisable disables authentication.
func (s *Server) AuthDisable(ctx context.Context, req *pb.AuthDisableRequest) (*pb.AuthDisableResponse, error) {
	resp, err := s.cache.Server.AuthDisable(ctx, &etcdserverpb.AuthDisableRequest{})
	if err != nil {
		return &pb.AuthDisableResponse{}, err
	}
	return &pb.AuthDisableResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// Authenticate processes an authenticate request.
func (s *Server) Authenticate(ctx context.Context, req *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	resp, err := s.cache.Server.Authenticate(ctx, &etcdserverpb.AuthenticateRequest{Name: req.Name, Password: req.Password})
	if err != nil {
		return &pb.AuthenticateResponse{}, err
	}
	return &pb.AuthenticateResponse{
		Header: s.convertHeader(resp.Header),
		Token:  resp.Token,
	}, err
}

// UserAdd adds a new user.
func (s *Server) UserAdd(ctx context.Context, req *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error) {
	resp, err := s.cache.Server.UserAdd(ctx, &etcdserverpb.AuthUserAddRequest{Name: req.Name, Password: req.Password})
	if err != nil {
		return &pb.AuthUserAddResponse{}, err
	}
	return &pb.AuthUserAddResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// UserGet gets detailed information for a user.
func (s *Server) UserGet(ctx context.Context, req *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	resp, err := s.cache.Server.UserGet(ctx, &etcdserverpb.AuthUserGetRequest{Name: req.Name})
	if err != nil {
		return &pb.AuthUserGetResponse{}, err
	}
	return &pb.AuthUserGetResponse{
		Header: s.convertHeader(resp.Header),
		Roles:  resp.Roles,
	}, err
}

// UserList gets a list of all users.
func (s *Server) UserList(ctx context.Context, req *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error) {
	resp, err := s.cache.Server.UserList(ctx, &etcdserverpb.AuthUserListRequest{})
	if err != nil {
		return &pb.AuthUserListResponse{}, err
	}

	return &pb.AuthUserListResponse{
		Header: s.convertHeader(resp.Header),
		Users:  resp.Users,
	}, err
}

// UserDelete deletes a specified user.
func (s *Server) UserDelete(ctx context.Context, req *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error) {
	if req == nil {
		return &pb.AuthUserDeleteResponse{}, errors.New("Unknown error")
	}

	resp, err := s.cache.Server.UserDelete(ctx, &etcdserverpb.AuthUserDeleteRequest{Name: req.Name})
	if err != nil {
		return &pb.AuthUserDeleteResponse{}, err
	}
	return &pb.AuthUserDeleteResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// UserChangePassword changes the password of a specified user.
func (s *Server) UserChangePassword(ctx context.Context, req *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error) {
	resp, err := s.cache.Server.UserChangePassword(ctx, &etcdserverpb.AuthUserChangePasswordRequest{Name: req.Name, Password: req.Password})
	if err != nil {
		return &pb.AuthUserChangePasswordResponse{}, err
	}
	return &pb.AuthUserChangePasswordResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// UserGrantRole grants a role to a specified user.
func (s *Server) UserGrantRole(ctx context.Context, req *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error) {
	resp, err := s.cache.Server.UserGrantRole(ctx, &etcdserverpb.AuthUserGrantRoleRequest{Role: req.Role, User: req.User})
	if err != nil {
		return &pb.AuthUserGrantRoleResponse{}, err
	}
	return &pb.AuthUserGrantRoleResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// UserRevokeRole revokes a role from a specified user.
func (s *Server) UserRevokeRole(ctx context.Context, req *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error) {
	resp, err := s.cache.Server.UserRevokeRole(ctx, &etcdserverpb.AuthUserRevokeRoleRequest{Name: req.Name, Role: req.Role})
	if err != nil {
		return &pb.AuthUserRevokeRoleResponse{}, err
	}
	return &pb.AuthUserRevokeRoleResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// RoleAdd adds a new role.
func (s *Server) RoleAdd(ctx context.Context, req *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error) {
	resp, err := s.cache.Server.RoleAdd(ctx, &etcdserverpb.AuthRoleAddRequest{Name: req.Name})
	if err != nil {
		return &pb.AuthRoleAddResponse{}, err
	}
	return &pb.AuthRoleAddResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// RoleGet gets detailed role information.
func (s *Server) RoleGet(ctx context.Context, req *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	resp, err := s.cache.Server.RoleGet(ctx, &etcdserverpb.AuthRoleGetRequest{Role: req.Role})
	if err != nil {
		return &pb.AuthRoleGetResponse{}, err
	}
	return &pb.AuthRoleGetResponse{
		Header: s.convertHeader(resp.Header),
		Perm:   s.convertPermissions(resp.Perm),
	}, err
}

// RoleList gets a list of all rolls.
func (s *Server) RoleList(ctx context.Context, req *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error) {
	resp, err := s.cache.Server.RoleList(ctx, &etcdserverpb.AuthRoleListRequest{})
	if err != nil {
		return &pb.AuthRoleListResponse{}, err
	}
	return &pb.AuthRoleListResponse{
		Header: s.convertHeader(resp.Header),
		Roles:  resp.Roles,
	}, err
}

// RoleDelete deletes a specified role.
func (s *Server) RoleDelete(ctx context.Context, req *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error) {
	resp, err := s.cache.Server.RoleDelete(ctx, &etcdserverpb.AuthRoleDeleteRequest{Role: req.Role})
	if err != nil {
		return &pb.AuthRoleDeleteResponse{}, err
	}
	return &pb.AuthRoleDeleteResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// RoleGrantPermission grants a permission of a specified key or range to a specified role.
func (s *Server) RoleGrantPermission(ctx context.Context, req *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error) {
	resp, err := s.cache.Server.RoleGrantPermission(ctx, &etcdserverpb.AuthRoleGrantPermissionRequest{Name: req.Name, Perm: s.convertPermission(req.Perm)})
	if err != nil {
		return &pb.AuthRoleGrantPermissionResponse{}, err
	}
	return &pb.AuthRoleGrantPermissionResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

// RoleRevokePermission revokes a permission of a specified key or range from a specified role.
func (s *Server) RoleRevokePermission(ctx context.Context, req *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error) {
	resp, err := s.cache.Server.RoleRevokePermission(ctx, &etcdserverpb.AuthRoleRevokePermissionRequest{Key: req.Key, RangeEnd: req.RangeEnd, Role: req.Role})
	if err != nil {
		return &pb.AuthRoleRevokePermissionResponse{}, err
	}
	return &pb.AuthRoleRevokePermissionResponse{
		Header: s.convertHeader(resp.Header),
	}, err
}

func (s *Server) convertHeader(h *etcdserverpb.ResponseHeader) *pb.ResponseHeader {
	if h == nil {
		return &pb.ResponseHeader{}
	}

	return &pb.ResponseHeader{
		ClusterId: h.ClusterId,
		MemberId:  h.MemberId,
		RaftTerm:  h.RaftTerm,
		Revision:  h.Revision,
	}
}

func (s *Server) convertPermission(p *pb.Permission) *authpb.Permission {
	return &authpb.Permission{
		Key:      p.Key,
		PermType: authpb.Permission_Type(p.PermType),
		RangeEnd: p.RangeEnd,
	}
}

func (s *Server) convertPermissions(p []*authpb.Permission) []*pb.Permission {
	if p == nil {
		return []*pb.Permission{}
	}

	perms := make([]*pb.Permission, len(p))
	for i, perm := range p {
		perms[i] = &pb.Permission{
			Key:      perm.Key,
			PermType: pb.Permission_Type(perm.PermType),
			RangeEnd: perm.RangeEnd,
		}
	}
	return perms
}
