// Code generated by protoc-gen-gogo.
// source: raft4si.proto
// DO NOT EDIT!

/*
	Package raft4si is a generated protocol buffer package.

	It is generated from these files:
		raft4si.proto

	It has these top-level messages:
		PBSnapshot
		PBPropose
		PBEntry
		PBLeadershipUpdateMessage
*/
package raft4si

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// wal saved snapshot struct
type PBSnapshot struct {
	Snapdata  []byte `protobuf:"bytes,1,opt,name=snapdata,proto3" json:"snapdata,omitempty"`
	HardState []byte `protobuf:"bytes,2,opt,name=hard_state,json=hardState,proto3" json:"hard_state,omitempty"`
}

func (m *PBSnapshot) Reset()                    { *m = PBSnapshot{} }
func (m *PBSnapshot) String() string            { return proto.CompactTextString(m) }
func (*PBSnapshot) ProtoMessage()               {}
func (*PBSnapshot) Descriptor() ([]byte, []int) { return fileDescriptorRaft4Si, []int{0} }

// Raft propose struct
type PBPropose struct {
	ProposeID uint64 `protobuf:"varint,2,opt,name=ProposeID,json=proposeID,proto3" json:"ProposeID,omitempty"`
	Request   []byte `protobuf:"bytes,1,opt,name=Request,json=request,proto3" json:"Request,omitempty"`
}

func (m *PBPropose) Reset()                    { *m = PBPropose{} }
func (m *PBPropose) String() string            { return proto.CompactTextString(m) }
func (*PBPropose) ProtoMessage()               {}
func (*PBPropose) Descriptor() ([]byte, []int) { return fileDescriptorRaft4Si, []int{1} }

// wal saved Entry struct
type PBEntry struct {
	Entry     []byte `protobuf:"bytes,1,opt,name=entry,proto3" json:"entry,omitempty"`
	HardState []byte `protobuf:"bytes,2,opt,name=hard_state,json=hardState,proto3" json:"hard_state,omitempty"`
}

func (m *PBEntry) Reset()                    { *m = PBEntry{} }
func (m *PBEntry) String() string            { return proto.CompactTextString(m) }
func (*PBEntry) ProtoMessage()               {}
func (*PBEntry) Descriptor() ([]byte, []int) { return fileDescriptorRaft4Si, []int{2} }

// send to service Message struct
type PBLeadershipUpdateMessage struct {
	LocalLeader bool `protobuf:"varint,1,opt,name=local_leader,json=localLeader,proto3" json:"local_leader,omitempty"`
}

func (m *PBLeadershipUpdateMessage) Reset()                    { *m = PBLeadershipUpdateMessage{} }
func (m *PBLeadershipUpdateMessage) String() string            { return proto.CompactTextString(m) }
func (*PBLeadershipUpdateMessage) ProtoMessage()               {}
func (*PBLeadershipUpdateMessage) Descriptor() ([]byte, []int) { return fileDescriptorRaft4Si, []int{3} }

func init() {
	proto.RegisterType((*PBSnapshot)(nil), "raft4si.PBSnapshot")
	proto.RegisterType((*PBPropose)(nil), "raft4si.PBPropose")
	proto.RegisterType((*PBEntry)(nil), "raft4si.PBEntry")
	proto.RegisterType((*PBLeadershipUpdateMessage)(nil), "raft4si.PBLeadershipUpdateMessage")
}
func (m *PBSnapshot) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *PBSnapshot) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Snapdata) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintRaft4Si(data, i, uint64(len(m.Snapdata)))
		i += copy(data[i:], m.Snapdata)
	}
	if len(m.HardState) > 0 {
		data[i] = 0x12
		i++
		i = encodeVarintRaft4Si(data, i, uint64(len(m.HardState)))
		i += copy(data[i:], m.HardState)
	}
	return i, nil
}

func (m *PBPropose) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *PBPropose) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Request) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintRaft4Si(data, i, uint64(len(m.Request)))
		i += copy(data[i:], m.Request)
	}
	if m.ProposeID != 0 {
		data[i] = 0x10
		i++
		i = encodeVarintRaft4Si(data, i, uint64(m.ProposeID))
	}
	return i, nil
}

func (m *PBEntry) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *PBEntry) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Entry) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintRaft4Si(data, i, uint64(len(m.Entry)))
		i += copy(data[i:], m.Entry)
	}
	if len(m.HardState) > 0 {
		data[i] = 0x12
		i++
		i = encodeVarintRaft4Si(data, i, uint64(len(m.HardState)))
		i += copy(data[i:], m.HardState)
	}
	return i, nil
}

func (m *PBLeadershipUpdateMessage) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *PBLeadershipUpdateMessage) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.LocalLeader {
		data[i] = 0x8
		i++
		if m.LocalLeader {
			data[i] = 1
		} else {
			data[i] = 0
		}
		i++
	}
	return i, nil
}

func encodeFixed64Raft4Si(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Raft4Si(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintRaft4Si(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *PBSnapshot) Size() (n int) {
	var l int
	_ = l
	l = len(m.Snapdata)
	if l > 0 {
		n += 1 + l + sovRaft4Si(uint64(l))
	}
	l = len(m.HardState)
	if l > 0 {
		n += 1 + l + sovRaft4Si(uint64(l))
	}
	return n
}

func (m *PBPropose) Size() (n int) {
	var l int
	_ = l
	l = len(m.Request)
	if l > 0 {
		n += 1 + l + sovRaft4Si(uint64(l))
	}
	if m.ProposeID != 0 {
		n += 1 + sovRaft4Si(uint64(m.ProposeID))
	}
	return n
}

func (m *PBEntry) Size() (n int) {
	var l int
	_ = l
	l = len(m.Entry)
	if l > 0 {
		n += 1 + l + sovRaft4Si(uint64(l))
	}
	l = len(m.HardState)
	if l > 0 {
		n += 1 + l + sovRaft4Si(uint64(l))
	}
	return n
}

func (m *PBLeadershipUpdateMessage) Size() (n int) {
	var l int
	_ = l
	if m.LocalLeader {
		n += 2
	}
	return n
}

func sovRaft4Si(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozRaft4Si(x uint64) (n int) {
	return sovRaft4Si(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *PBSnapshot) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaft4Si
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PBSnapshot: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PBSnapshot: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Snapdata", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRaft4Si
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Snapdata = append(m.Snapdata[:0], data[iNdEx:postIndex]...)
			if m.Snapdata == nil {
				m.Snapdata = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field HardState", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRaft4Si
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.HardState = append(m.HardState[:0], data[iNdEx:postIndex]...)
			if m.HardState == nil {
				m.HardState = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaft4Si(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaft4Si
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *PBPropose) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaft4Si
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PBPropose: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PBPropose: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Request", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRaft4Si
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Request = append(m.Request[:0], data[iNdEx:postIndex]...)
			if m.Request == nil {
				m.Request = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProposeID", wireType)
			}
			m.ProposeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.ProposeID |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipRaft4Si(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaft4Si
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *PBEntry) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaft4Si
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PBEntry: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PBEntry: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Entry", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRaft4Si
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Entry = append(m.Entry[:0], data[iNdEx:postIndex]...)
			if m.Entry == nil {
				m.Entry = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field HardState", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthRaft4Si
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.HardState = append(m.HardState[:0], data[iNdEx:postIndex]...)
			if m.HardState == nil {
				m.HardState = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipRaft4Si(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaft4Si
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *PBLeadershipUpdateMessage) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowRaft4Si
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PBLeadershipUpdateMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PBLeadershipUpdateMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LocalLeader", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.LocalLeader = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipRaft4Si(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthRaft4Si
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipRaft4Si(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowRaft4Si
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowRaft4Si
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthRaft4Si
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowRaft4Si
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipRaft4Si(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthRaft4Si = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowRaft4Si   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("raft4si.proto", fileDescriptorRaft4Si) }

var fileDescriptorRaft4Si = []byte{
	// 247 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x90, 0xb1, 0x4a, 0xc4, 0x40,
	0x10, 0x86, 0x59, 0x51, 0x73, 0x19, 0x63, 0xb3, 0x5c, 0x11, 0x0f, 0x0d, 0x9a, 0xca, 0x4a, 0x0b,
	0xad, 0xaf, 0x58, 0x15, 0x11, 0x14, 0x96, 0x1c, 0xd6, 0xc7, 0x68, 0xc6, 0xe4, 0x20, 0x64, 0xd7,
	0x9d, 0xb5, 0xf0, 0x0d, 0x2d, 0x7d, 0x04, 0xc9, 0x93, 0xc8, 0x6e, 0xa2, 0xed, 0x75, 0xf3, 0x7f,
	0x33, 0xfb, 0xf1, 0xb3, 0x70, 0xe8, 0xf0, 0xcd, 0x5f, 0xf3, 0xe6, 0xc2, 0x3a, 0xe3, 0x8d, 0x4c,
	0xa6, 0xb8, 0x98, 0x37, 0xa6, 0x31, 0x91, 0x5d, 0x86, 0x69, 0x5c, 0x97, 0xf7, 0x00, 0x5a, 0xad,
	0x7a, 0xb4, 0xdc, 0x1a, 0x2f, 0x17, 0x30, 0xe3, 0x1e, 0x6d, 0x8d, 0x1e, 0x73, 0x71, 0x2a, 0xce,
	0xb3, 0xea, 0x3f, 0xcb, 0x13, 0x80, 0x16, 0x5d, 0xbd, 0x66, 0x8f, 0x9e, 0xf2, 0x9d, 0xb8, 0x4d,
	0x03, 0x59, 0x05, 0x50, 0xde, 0x40, 0xaa, 0x95, 0x76, 0xc6, 0x1a, 0x26, 0x99, 0x43, 0x52, 0xd1,
	0xfb, 0x07, 0xb1, 0x9f, 0x34, 0x89, 0x1b, 0xa3, 0x3c, 0x86, 0x74, 0x3a, 0x7a, 0xb8, 0x8d, 0x92,
	0xdd, 0x2a, 0xb5, 0x7f, 0xa0, 0x5c, 0x42, 0xa2, 0xd5, 0x5d, 0xef, 0xdd, 0xa7, 0x9c, 0xc3, 0x1e,
	0x85, 0x61, 0x12, 0x8c, 0x61, 0x5b, 0x89, 0x25, 0x1c, 0x69, 0xf5, 0x48, 0x58, 0x93, 0xe3, 0x76,
	0x63, 0x9f, 0x43, 0x73, 0x7a, 0x22, 0x66, 0x6c, 0x48, 0x9e, 0x41, 0xd6, 0x99, 0x57, 0xec, 0xd6,
	0x5d, 0x3c, 0x88, 0xe2, 0x59, 0x75, 0x10, 0xd9, 0xf8, 0x46, 0x65, 0x5f, 0x43, 0x21, 0xbe, 0x87,
	0x42, 0xfc, 0x0c, 0x85, 0x78, 0xd9, 0x8f, 0x5f, 0x74, 0xf5, 0x1b, 0x00, 0x00, 0xff, 0xff, 0x4d,
	0x9b, 0xc1, 0x64, 0x52, 0x01, 0x00, 0x00,
}
