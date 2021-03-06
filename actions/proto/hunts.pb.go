// Code generated by protoc-gen-go. DO NOT EDIT.
// source: hunts.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ForemanCheckin struct {
	LastHuntTimestamp    uint64   `protobuf:"varint,1,opt,name=last_hunt_timestamp,json=lastHuntTimestamp,proto3" json:"last_hunt_timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ForemanCheckin) Reset()         { *m = ForemanCheckin{} }
func (m *ForemanCheckin) String() string { return proto.CompactTextString(m) }
func (*ForemanCheckin) ProtoMessage()    {}
func (*ForemanCheckin) Descriptor() ([]byte, []int) {
	return fileDescriptor_hunts_dbd2d0544e5020f3, []int{0}
}
func (m *ForemanCheckin) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ForemanCheckin.Unmarshal(m, b)
}
func (m *ForemanCheckin) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ForemanCheckin.Marshal(b, m, deterministic)
}
func (dst *ForemanCheckin) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ForemanCheckin.Merge(dst, src)
}
func (m *ForemanCheckin) XXX_Size() int {
	return xxx_messageInfo_ForemanCheckin.Size(m)
}
func (m *ForemanCheckin) XXX_DiscardUnknown() {
	xxx_messageInfo_ForemanCheckin.DiscardUnknown(m)
}

var xxx_messageInfo_ForemanCheckin proto.InternalMessageInfo

func (m *ForemanCheckin) GetLastHuntTimestamp() uint64 {
	if m != nil {
		return m.LastHuntTimestamp
	}
	return 0
}

func init() {
	proto.RegisterType((*ForemanCheckin)(nil), "proto.ForemanCheckin")
}

func init() { proto.RegisterFile("hunts.proto", fileDescriptor_hunts_dbd2d0544e5020f3) }

var fileDescriptor_hunts_dbd2d0544e5020f3 = []byte{
	// 102 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xce, 0x28, 0xcd, 0x2b,
	0x29, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x4a, 0x0e, 0x5c, 0x7c, 0x6e,
	0xf9, 0x45, 0xa9, 0xb9, 0x89, 0x79, 0xce, 0x19, 0xa9, 0xc9, 0xd9, 0x99, 0x79, 0x42, 0x7a, 0x5c,
	0xc2, 0x39, 0x89, 0xc5, 0x25, 0xf1, 0x20, 0xc5, 0xf1, 0x25, 0x99, 0xb9, 0xa9, 0xc5, 0x25, 0x89,
	0xb9, 0x05, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x2c, 0x41, 0x82, 0x20, 0x29, 0x8f, 0xd2, 0xbc, 0x92,
	0x10, 0x98, 0x44, 0x12, 0x1b, 0xd8, 0x20, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x9f, 0x0a,
	0x0e, 0x00, 0x5e, 0x00, 0x00, 0x00,
}
