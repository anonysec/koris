package knodepb

import protoreflect "google.golang.org/protobuf/reflect/protoreflect"

// GenerateClientCertRequest is the request for generating an OpenVPN client certificate.
type GenerateClientCertRequest struct {
	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
}

func (x *GenerateClientCertRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *GenerateClientCertRequest) Reset()         {}
func (x *GenerateClientCertRequest) String() string { return x.Username }
func (x *GenerateClientCertRequest) ProtoMessage()  {}

func (x *GenerateClientCertRequest) ProtoReflect() protoreflect.Message { return nil }

// GenerateClientCertResponse contains the generated cert, key, and CA.
type GenerateClientCertResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	CertPem string `protobuf:"bytes,2,opt,name=cert_pem,proto3" json:"cert_pem,omitempty"`
	KeyPem  string `protobuf:"bytes,3,opt,name=key_pem,proto3" json:"key_pem,omitempty"`
	CaPem   string `protobuf:"bytes,4,opt,name=ca_pem,proto3" json:"ca_pem,omitempty"`
	Message string `protobuf:"bytes,5,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *GenerateClientCertResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *GenerateClientCertResponse) GetCertPem() string {
	if x != nil {
		return x.CertPem
	}
	return ""
}

func (x *GenerateClientCertResponse) GetKeyPem() string {
	if x != nil {
		return x.KeyPem
	}
	return ""
}

func (x *GenerateClientCertResponse) GetCaPem() string {
	if x != nil {
		return x.CaPem
	}
	return ""
}

func (x *GenerateClientCertResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *GenerateClientCertResponse) Reset()         {}
func (x *GenerateClientCertResponse) String() string { return x.Message }
func (x *GenerateClientCertResponse) ProtoMessage()  {}

func (x *GenerateClientCertResponse) ProtoReflect() protoreflect.Message { return nil }
