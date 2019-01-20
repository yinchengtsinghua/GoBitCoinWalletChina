
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//由Protoc Gen Go生成的代码。不要编辑。
//来源：api.proto

/*
包walletrpc是生成的协议缓冲包。

它由以下文件生成：
 原始的

它包含以下顶级消息：
 版本请求
 版本响应
 交易详细信息
 封锁细节
 会计平衡
 平请求
 平响
 网络请求
 网络响应
 帐户号码请求
 账号响应
 会计要求
 账户响应
 重命名帐户请求
 重命名帐户响应
 下一个计数请求
 下一个计数响应
 下一步任务请求
 下一步响应
 导入私钥请求
 导入私钥响应
 平衡器
 平衡响应
 获取事务请求
 获取交易响应
 更改密码请求
 更改密码响应
 资金交易请求
 资金交易响应
 签署交易请求
 签名事务响应
 发布事务处理请求
 发布事务响应
 事务通知请求
 事务通知响应
 Spentness通知请求
 Spentness通知响应
 会计通知请求
 会计通知响应
 创建墙请求
 创建墙响应
 开放式墙壁测试
 
 封闭墙测试
 封闭墙响应
 墙存在请求
 
 开始协商请求
 启动共识响应
**/

package walletrpc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

//引用导入以禁止错误（如果未使用）。
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

//这是一个编译时断言，以确保生成的文件
//与正在编译的proto包兼容。
//此行的编译错误可能意味着您的
//需要更新proto包。
const _ = proto.ProtoPackageIsVersion2 //请升级proto包

type NextAddressRequest_Kind int32

const (
	NextAddressRequest_BIP0044_EXTERNAL NextAddressRequest_Kind = 0
	NextAddressRequest_BIP0044_INTERNAL NextAddressRequest_Kind = 1
)

var NextAddressRequest_Kind_name = map[int32]string{
	0: "BIP0044_EXTERNAL",
	1: "BIP0044_INTERNAL",
}
var NextAddressRequest_Kind_value = map[string]int32{
	"BIP0044_EXTERNAL": 0,
	"BIP0044_INTERNAL": 1,
}

func (x NextAddressRequest_Kind) String() string {
	return proto.EnumName(NextAddressRequest_Kind_name, int32(x))
}
func (NextAddressRequest_Kind) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{17, 0} }

type ChangePassphraseRequest_Key int32

const (
	ChangePassphraseRequest_PRIVATE ChangePassphraseRequest_Key = 0
	ChangePassphraseRequest_PUBLIC  ChangePassphraseRequest_Key = 1
)

var ChangePassphraseRequest_Key_name = map[int32]string{
	0: "PRIVATE",
	1: "PUBLIC",
}
var ChangePassphraseRequest_Key_value = map[string]int32{
	"PRIVATE": 0,
	"PUBLIC":  1,
}

func (x ChangePassphraseRequest_Key) String() string {
	return proto.EnumName(ChangePassphraseRequest_Key_name, int32(x))
}
func (ChangePassphraseRequest_Key) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor0, []int{25, 0}
}

type VersionRequest struct {
}

func (m *VersionRequest) Reset()                    { *m = VersionRequest{} }
func (m *VersionRequest) String() string            { return proto.CompactTextString(m) }
func (*VersionRequest) ProtoMessage()               {}
func (*VersionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type VersionResponse struct {
	VersionString string `protobuf:"bytes,1,opt,name=version_string,json=versionString" json:"version_string,omitempty"`
	Major         uint32 `protobuf:"varint,2,opt,name=major" json:"major,omitempty"`
	Minor         uint32 `protobuf:"varint,3,opt,name=minor" json:"minor,omitempty"`
	Patch         uint32 `protobuf:"varint,4,opt,name=patch" json:"patch,omitempty"`
	Prerelease    string `protobuf:"bytes,5,opt,name=prerelease" json:"prerelease,omitempty"`
	BuildMetadata string `protobuf:"bytes,6,opt,name=build_metadata,json=buildMetadata" json:"build_metadata,omitempty"`
}

func (m *VersionResponse) Reset()                    { *m = VersionResponse{} }
func (m *VersionResponse) String() string            { return proto.CompactTextString(m) }
func (*VersionResponse) ProtoMessage()               {}
func (*VersionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *VersionResponse) GetVersionString() string {
	if m != nil {
		return m.VersionString
	}
	return ""
}

func (m *VersionResponse) GetMajor() uint32 {
	if m != nil {
		return m.Major
	}
	return 0
}

func (m *VersionResponse) GetMinor() uint32 {
	if m != nil {
		return m.Minor
	}
	return 0
}

func (m *VersionResponse) GetPatch() uint32 {
	if m != nil {
		return m.Patch
	}
	return 0
}

func (m *VersionResponse) GetPrerelease() string {
	if m != nil {
		return m.Prerelease
	}
	return ""
}

func (m *VersionResponse) GetBuildMetadata() string {
	if m != nil {
		return m.BuildMetadata
	}
	return ""
}

type TransactionDetails struct {
	Hash        []byte                       `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Transaction []byte                       `protobuf:"bytes,2,opt,name=transaction,proto3" json:"transaction,omitempty"`
	Debits      []*TransactionDetails_Input  `protobuf:"bytes,3,rep,name=debits" json:"debits,omitempty"`
	Credits     []*TransactionDetails_Output `protobuf:"bytes,4,rep,name=credits" json:"credits,omitempty"`
	Fee         int64                        `protobuf:"varint,5,opt,name=fee" json:"fee,omitempty"`
	Timestamp   int64                        `protobuf:"varint,6,opt,name=timestamp" json:"timestamp,omitempty"`
}

func (m *TransactionDetails) Reset()                    { *m = TransactionDetails{} }
func (m *TransactionDetails) String() string            { return proto.CompactTextString(m) }
func (*TransactionDetails) ProtoMessage()               {}
func (*TransactionDetails) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *TransactionDetails) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *TransactionDetails) GetTransaction() []byte {
	if m != nil {
		return m.Transaction
	}
	return nil
}

func (m *TransactionDetails) GetDebits() []*TransactionDetails_Input {
	if m != nil {
		return m.Debits
	}
	return nil
}

func (m *TransactionDetails) GetCredits() []*TransactionDetails_Output {
	if m != nil {
		return m.Credits
	}
	return nil
}

func (m *TransactionDetails) GetFee() int64 {
	if m != nil {
		return m.Fee
	}
	return 0
}

func (m *TransactionDetails) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

type TransactionDetails_Input struct {
	Index           uint32 `protobuf:"varint,1,opt,name=index" json:"index,omitempty"`
	PreviousAccount uint32 `protobuf:"varint,2,opt,name=previous_account,json=previousAccount" json:"previous_account,omitempty"`
	PreviousAmount  int64  `protobuf:"varint,3,opt,name=previous_amount,json=previousAmount" json:"previous_amount,omitempty"`
}

func (m *TransactionDetails_Input) Reset()                    { *m = TransactionDetails_Input{} }
func (m *TransactionDetails_Input) String() string            { return proto.CompactTextString(m) }
func (*TransactionDetails_Input) ProtoMessage()               {}
func (*TransactionDetails_Input) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

func (m *TransactionDetails_Input) GetIndex() uint32 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *TransactionDetails_Input) GetPreviousAccount() uint32 {
	if m != nil {
		return m.PreviousAccount
	}
	return 0
}

func (m *TransactionDetails_Input) GetPreviousAmount() int64 {
	if m != nil {
		return m.PreviousAmount
	}
	return 0
}

type TransactionDetails_Output struct {
	Index    uint32 `protobuf:"varint,1,opt,name=index" json:"index,omitempty"`
	Account  uint32 `protobuf:"varint,2,opt,name=account" json:"account,omitempty"`
	Internal bool   `protobuf:"varint,3,opt,name=internal" json:"internal,omitempty"`
}

func (m *TransactionDetails_Output) Reset()                    { *m = TransactionDetails_Output{} }
func (m *TransactionDetails_Output) String() string            { return proto.CompactTextString(m) }
func (*TransactionDetails_Output) ProtoMessage()               {}
func (*TransactionDetails_Output) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 1} }

func (m *TransactionDetails_Output) GetIndex() uint32 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *TransactionDetails_Output) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *TransactionDetails_Output) GetInternal() bool {
	if m != nil {
		return m.Internal
	}
	return false
}

type BlockDetails struct {
	Hash         []byte                `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Height       int32                 `protobuf:"varint,2,opt,name=height" json:"height,omitempty"`
	Timestamp    int64                 `protobuf:"varint,3,opt,name=timestamp" json:"timestamp,omitempty"`
	Transactions []*TransactionDetails `protobuf:"bytes,4,rep,name=transactions" json:"transactions,omitempty"`
}

func (m *BlockDetails) Reset()                    { *m = BlockDetails{} }
func (m *BlockDetails) String() string            { return proto.CompactTextString(m) }
func (*BlockDetails) ProtoMessage()               {}
func (*BlockDetails) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *BlockDetails) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *BlockDetails) GetHeight() int32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *BlockDetails) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *BlockDetails) GetTransactions() []*TransactionDetails {
	if m != nil {
		return m.Transactions
	}
	return nil
}

type AccountBalance struct {
	Account      uint32 `protobuf:"varint,1,opt,name=account" json:"account,omitempty"`
	TotalBalance int64  `protobuf:"varint,2,opt,name=total_balance,json=totalBalance" json:"total_balance,omitempty"`
}

func (m *AccountBalance) Reset()                    { *m = AccountBalance{} }
func (m *AccountBalance) String() string            { return proto.CompactTextString(m) }
func (*AccountBalance) ProtoMessage()               {}
func (*AccountBalance) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *AccountBalance) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *AccountBalance) GetTotalBalance() int64 {
	if m != nil {
		return m.TotalBalance
	}
	return 0
}

type PingRequest struct {
}

func (m *PingRequest) Reset()                    { *m = PingRequest{} }
func (m *PingRequest) String() string            { return proto.CompactTextString(m) }
func (*PingRequest) ProtoMessage()               {}
func (*PingRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

type PingResponse struct {
}

func (m *PingResponse) Reset()                    { *m = PingResponse{} }
func (m *PingResponse) String() string            { return proto.CompactTextString(m) }
func (*PingResponse) ProtoMessage()               {}
func (*PingResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

type NetworkRequest struct {
}

func (m *NetworkRequest) Reset()                    { *m = NetworkRequest{} }
func (m *NetworkRequest) String() string            { return proto.CompactTextString(m) }
func (*NetworkRequest) ProtoMessage()               {}
func (*NetworkRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

type NetworkResponse struct {
	ActiveNetwork uint32 `protobuf:"varint,1,opt,name=active_network,json=activeNetwork" json:"active_network,omitempty"`
}

func (m *NetworkResponse) Reset()                    { *m = NetworkResponse{} }
func (m *NetworkResponse) String() string            { return proto.CompactTextString(m) }
func (*NetworkResponse) ProtoMessage()               {}
func (*NetworkResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *NetworkResponse) GetActiveNetwork() uint32 {
	if m != nil {
		return m.ActiveNetwork
	}
	return 0
}

type AccountNumberRequest struct {
	AccountName string `protobuf:"bytes,1,opt,name=account_name,json=accountName" json:"account_name,omitempty"`
}

func (m *AccountNumberRequest) Reset()                    { *m = AccountNumberRequest{} }
func (m *AccountNumberRequest) String() string            { return proto.CompactTextString(m) }
func (*AccountNumberRequest) ProtoMessage()               {}
func (*AccountNumberRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (m *AccountNumberRequest) GetAccountName() string {
	if m != nil {
		return m.AccountName
	}
	return ""
}

type AccountNumberResponse struct {
	AccountNumber uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
}

func (m *AccountNumberResponse) Reset()                    { *m = AccountNumberResponse{} }
func (m *AccountNumberResponse) String() string            { return proto.CompactTextString(m) }
func (*AccountNumberResponse) ProtoMessage()               {}
func (*AccountNumberResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *AccountNumberResponse) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

type AccountsRequest struct {
}

func (m *AccountsRequest) Reset()                    { *m = AccountsRequest{} }
func (m *AccountsRequest) String() string            { return proto.CompactTextString(m) }
func (*AccountsRequest) ProtoMessage()               {}
func (*AccountsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

type AccountsResponse struct {
	Accounts           []*AccountsResponse_Account `protobuf:"bytes,1,rep,name=accounts" json:"accounts,omitempty"`
	CurrentBlockHash   []byte                      `protobuf:"bytes,2,opt,name=current_block_hash,json=currentBlockHash,proto3" json:"current_block_hash,omitempty"`
	CurrentBlockHeight int32                       `protobuf:"varint,3,opt,name=current_block_height,json=currentBlockHeight" json:"current_block_height,omitempty"`
}

func (m *AccountsResponse) Reset()                    { *m = AccountsResponse{} }
func (m *AccountsResponse) String() string            { return proto.CompactTextString(m) }
func (*AccountsResponse) ProtoMessage()               {}
func (*AccountsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12} }

func (m *AccountsResponse) GetAccounts() []*AccountsResponse_Account {
	if m != nil {
		return m.Accounts
	}
	return nil
}

func (m *AccountsResponse) GetCurrentBlockHash() []byte {
	if m != nil {
		return m.CurrentBlockHash
	}
	return nil
}

func (m *AccountsResponse) GetCurrentBlockHeight() int32 {
	if m != nil {
		return m.CurrentBlockHeight
	}
	return 0
}

type AccountsResponse_Account struct {
	AccountNumber    uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
	AccountName      string `protobuf:"bytes,2,opt,name=account_name,json=accountName" json:"account_name,omitempty"`
	TotalBalance     int64  `protobuf:"varint,3,opt,name=total_balance,json=totalBalance" json:"total_balance,omitempty"`
	ExternalKeyCount uint32 `protobuf:"varint,4,opt,name=external_key_count,json=externalKeyCount" json:"external_key_count,omitempty"`
	InternalKeyCount uint32 `protobuf:"varint,5,opt,name=internal_key_count,json=internalKeyCount" json:"internal_key_count,omitempty"`
	ImportedKeyCount uint32 `protobuf:"varint,6,opt,name=imported_key_count,json=importedKeyCount" json:"imported_key_count,omitempty"`
}

func (m *AccountsResponse_Account) Reset()                    { *m = AccountsResponse_Account{} }
func (m *AccountsResponse_Account) String() string            { return proto.CompactTextString(m) }
func (*AccountsResponse_Account) ProtoMessage()               {}
func (*AccountsResponse_Account) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12, 0} }

func (m *AccountsResponse_Account) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

func (m *AccountsResponse_Account) GetAccountName() string {
	if m != nil {
		return m.AccountName
	}
	return ""
}

func (m *AccountsResponse_Account) GetTotalBalance() int64 {
	if m != nil {
		return m.TotalBalance
	}
	return 0
}

func (m *AccountsResponse_Account) GetExternalKeyCount() uint32 {
	if m != nil {
		return m.ExternalKeyCount
	}
	return 0
}

func (m *AccountsResponse_Account) GetInternalKeyCount() uint32 {
	if m != nil {
		return m.InternalKeyCount
	}
	return 0
}

func (m *AccountsResponse_Account) GetImportedKeyCount() uint32 {
	if m != nil {
		return m.ImportedKeyCount
	}
	return 0
}

type RenameAccountRequest struct {
	AccountNumber uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
	NewName       string `protobuf:"bytes,2,opt,name=new_name,json=newName" json:"new_name,omitempty"`
}

func (m *RenameAccountRequest) Reset()                    { *m = RenameAccountRequest{} }
func (m *RenameAccountRequest) String() string            { return proto.CompactTextString(m) }
func (*RenameAccountRequest) ProtoMessage()               {}
func (*RenameAccountRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{13} }

func (m *RenameAccountRequest) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

func (m *RenameAccountRequest) GetNewName() string {
	if m != nil {
		return m.NewName
	}
	return ""
}

type RenameAccountResponse struct {
}

func (m *RenameAccountResponse) Reset()                    { *m = RenameAccountResponse{} }
func (m *RenameAccountResponse) String() string            { return proto.CompactTextString(m) }
func (*RenameAccountResponse) ProtoMessage()               {}
func (*RenameAccountResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{14} }

type NextAccountRequest struct {
	Passphrase  []byte `protobuf:"bytes,1,opt,name=passphrase,proto3" json:"passphrase,omitempty"`
	AccountName string `protobuf:"bytes,2,opt,name=account_name,json=accountName" json:"account_name,omitempty"`
}

func (m *NextAccountRequest) Reset()                    { *m = NextAccountRequest{} }
func (m *NextAccountRequest) String() string            { return proto.CompactTextString(m) }
func (*NextAccountRequest) ProtoMessage()               {}
func (*NextAccountRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{15} }

func (m *NextAccountRequest) GetPassphrase() []byte {
	if m != nil {
		return m.Passphrase
	}
	return nil
}

func (m *NextAccountRequest) GetAccountName() string {
	if m != nil {
		return m.AccountName
	}
	return ""
}

type NextAccountResponse struct {
	AccountNumber uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
}

func (m *NextAccountResponse) Reset()                    { *m = NextAccountResponse{} }
func (m *NextAccountResponse) String() string            { return proto.CompactTextString(m) }
func (*NextAccountResponse) ProtoMessage()               {}
func (*NextAccountResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{16} }

func (m *NextAccountResponse) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

type NextAddressRequest struct {
	Account uint32                  `protobuf:"varint,1,opt,name=account" json:"account,omitempty"`
	Kind    NextAddressRequest_Kind `protobuf:"varint,2,opt,name=kind,enum=walletrpc.NextAddressRequest_Kind" json:"kind,omitempty"`
}

func (m *NextAddressRequest) Reset()                    { *m = NextAddressRequest{} }
func (m *NextAddressRequest) String() string            { return proto.CompactTextString(m) }
func (*NextAddressRequest) ProtoMessage()               {}
func (*NextAddressRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{17} }

func (m *NextAddressRequest) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *NextAddressRequest) GetKind() NextAddressRequest_Kind {
	if m != nil {
		return m.Kind
	}
	return NextAddressRequest_BIP0044_EXTERNAL
}

type NextAddressResponse struct {
	Address string `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
}

func (m *NextAddressResponse) Reset()                    { *m = NextAddressResponse{} }
func (m *NextAddressResponse) String() string            { return proto.CompactTextString(m) }
func (*NextAddressResponse) ProtoMessage()               {}
func (*NextAddressResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{18} }

func (m *NextAddressResponse) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type ImportPrivateKeyRequest struct {
	Passphrase    []byte `protobuf:"bytes,1,opt,name=passphrase,proto3" json:"passphrase,omitempty"`
	Account       uint32 `protobuf:"varint,2,opt,name=account" json:"account,omitempty"`
	PrivateKeyWif string `protobuf:"bytes,3,opt,name=private_key_wif,json=privateKeyWif" json:"private_key_wif,omitempty"`
	Rescan        bool   `protobuf:"varint,4,opt,name=rescan" json:"rescan,omitempty"`
}

func (m *ImportPrivateKeyRequest) Reset()                    { *m = ImportPrivateKeyRequest{} }
func (m *ImportPrivateKeyRequest) String() string            { return proto.CompactTextString(m) }
func (*ImportPrivateKeyRequest) ProtoMessage()               {}
func (*ImportPrivateKeyRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{19} }

func (m *ImportPrivateKeyRequest) GetPassphrase() []byte {
	if m != nil {
		return m.Passphrase
	}
	return nil
}

func (m *ImportPrivateKeyRequest) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *ImportPrivateKeyRequest) GetPrivateKeyWif() string {
	if m != nil {
		return m.PrivateKeyWif
	}
	return ""
}

func (m *ImportPrivateKeyRequest) GetRescan() bool {
	if m != nil {
		return m.Rescan
	}
	return false
}

type ImportPrivateKeyResponse struct {
}

func (m *ImportPrivateKeyResponse) Reset()                    { *m = ImportPrivateKeyResponse{} }
func (m *ImportPrivateKeyResponse) String() string            { return proto.CompactTextString(m) }
func (*ImportPrivateKeyResponse) ProtoMessage()               {}
func (*ImportPrivateKeyResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{20} }

type BalanceRequest struct {
	AccountNumber         uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
	RequiredConfirmations int32  `protobuf:"varint,2,opt,name=required_confirmations,json=requiredConfirmations" json:"required_confirmations,omitempty"`
}

func (m *BalanceRequest) Reset()                    { *m = BalanceRequest{} }
func (m *BalanceRequest) String() string            { return proto.CompactTextString(m) }
func (*BalanceRequest) ProtoMessage()               {}
func (*BalanceRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{21} }

func (m *BalanceRequest) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

func (m *BalanceRequest) GetRequiredConfirmations() int32 {
	if m != nil {
		return m.RequiredConfirmations
	}
	return 0
}

type BalanceResponse struct {
	Total          int64 `protobuf:"varint,1,opt,name=total" json:"total,omitempty"`
	Spendable      int64 `protobuf:"varint,2,opt,name=spendable" json:"spendable,omitempty"`
	ImmatureReward int64 `protobuf:"varint,3,opt,name=immature_reward,json=immatureReward" json:"immature_reward,omitempty"`
}

func (m *BalanceResponse) Reset()                    { *m = BalanceResponse{} }
func (m *BalanceResponse) String() string            { return proto.CompactTextString(m) }
func (*BalanceResponse) ProtoMessage()               {}
func (*BalanceResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{22} }

func (m *BalanceResponse) GetTotal() int64 {
	if m != nil {
		return m.Total
	}
	return 0
}

func (m *BalanceResponse) GetSpendable() int64 {
	if m != nil {
		return m.Spendable
	}
	return 0
}

func (m *BalanceResponse) GetImmatureReward() int64 {
	if m != nil {
		return m.ImmatureReward
	}
	return 0
}

type GetTransactionsRequest struct {
//可选地指定要从中开始的起始块，包括所有事务。
//可以指定起始块哈希或高度，但不能同时指定两者。
//如果指定了块高度且为负，则绝对值将成为
//要包含的最后一个块。也就是说，如果当前链的高度为1000，并且起始块
//高度为-3，将为块998、999和1000创建事务通知。
//如果两个选项都被排除，则会为自
//
	StartingBlockHash   []byte `protobuf:"bytes,1,opt,name=starting_block_hash,json=startingBlockHash,proto3" json:"starting_block_hash,omitempty"`
	StartingBlockHeight int32  `protobuf:"zigzag32,2,opt,name=starting_block_height,json=startingBlockHeight" json:"starting_block_height,omitempty"`
//可选地指定事务结果可能出现的最后一个块。
//可以指定结束块哈希或高度，但不能同时指定两者。
//如果两者都被排除，则为所有交易创建交易结果。
//通过最佳块，并包括所有未链接的事务。
	EndingBlockHash   []byte `protobuf:"bytes,3,opt,name=ending_block_hash,json=endingBlockHash,proto3" json:"ending_block_hash,omitempty"`
	EndingBlockHeight int32  `protobuf:"varint,4,opt,name=ending_block_height,json=endingBlockHeight" json:"ending_block_height,omitempty"`
//如果存在的话，至少包括这么多最新的事务。
//指定结束块哈希时不能使用。
//
//TODO:在spec以某种方式将其添加回之前移除。
	MinimumRecentTransactions int32 `protobuf:"varint,5,opt,name=minimum_recent_transactions,json=minimumRecentTransactions" json:"minimum_recent_transactions,omitempty"`
}

func (m *GetTransactionsRequest) Reset()                    { *m = GetTransactionsRequest{} }
func (m *GetTransactionsRequest) String() string            { return proto.CompactTextString(m) }
func (*GetTransactionsRequest) ProtoMessage()               {}
func (*GetTransactionsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{23} }

func (m *GetTransactionsRequest) GetStartingBlockHash() []byte {
	if m != nil {
		return m.StartingBlockHash
	}
	return nil
}

func (m *GetTransactionsRequest) GetStartingBlockHeight() int32 {
	if m != nil {
		return m.StartingBlockHeight
	}
	return 0
}

func (m *GetTransactionsRequest) GetEndingBlockHash() []byte {
	if m != nil {
		return m.EndingBlockHash
	}
	return nil
}

func (m *GetTransactionsRequest) GetEndingBlockHeight() int32 {
	if m != nil {
		return m.EndingBlockHeight
	}
	return 0
}

func (m *GetTransactionsRequest) GetMinimumRecentTransactions() int32 {
	if m != nil {
		return m.MinimumRecentTransactions
	}
	return 0
}

type GetTransactionsResponse struct {
	MinedTransactions   []*BlockDetails       `protobuf:"bytes,1,rep,name=mined_transactions,json=minedTransactions" json:"mined_transactions,omitempty"`
	UnminedTransactions []*TransactionDetails `protobuf:"bytes,2,rep,name=unmined_transactions,json=unminedTransactions" json:"unmined_transactions,omitempty"`
}

func (m *GetTransactionsResponse) Reset()                    { *m = GetTransactionsResponse{} }
func (m *GetTransactionsResponse) String() string            { return proto.CompactTextString(m) }
func (*GetTransactionsResponse) ProtoMessage()               {}
func (*GetTransactionsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{24} }

func (m *GetTransactionsResponse) GetMinedTransactions() []*BlockDetails {
	if m != nil {
		return m.MinedTransactions
	}
	return nil
}

func (m *GetTransactionsResponse) GetUnminedTransactions() []*TransactionDetails {
	if m != nil {
		return m.UnminedTransactions
	}
	return nil
}

type ChangePassphraseRequest struct {
	Key           ChangePassphraseRequest_Key `protobuf:"varint,1,opt,name=key,enum=walletrpc.ChangePassphraseRequest_Key" json:"key,omitempty"`
	OldPassphrase []byte                      `protobuf:"bytes,2,opt,name=old_passphrase,json=oldPassphrase,proto3" json:"old_passphrase,omitempty"`
	NewPassphrase []byte                      `protobuf:"bytes,3,opt,name=new_passphrase,json=newPassphrase,proto3" json:"new_passphrase,omitempty"`
}

func (m *ChangePassphraseRequest) Reset()                    { *m = ChangePassphraseRequest{} }
func (m *ChangePassphraseRequest) String() string            { return proto.CompactTextString(m) }
func (*ChangePassphraseRequest) ProtoMessage()               {}
func (*ChangePassphraseRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{25} }

func (m *ChangePassphraseRequest) GetKey() ChangePassphraseRequest_Key {
	if m != nil {
		return m.Key
	}
	return ChangePassphraseRequest_PRIVATE
}

func (m *ChangePassphraseRequest) GetOldPassphrase() []byte {
	if m != nil {
		return m.OldPassphrase
	}
	return nil
}

func (m *ChangePassphraseRequest) GetNewPassphrase() []byte {
	if m != nil {
		return m.NewPassphrase
	}
	return nil
}

type ChangePassphraseResponse struct {
}

func (m *ChangePassphraseResponse) Reset()                    { *m = ChangePassphraseResponse{} }
func (m *ChangePassphraseResponse) String() string            { return proto.CompactTextString(m) }
func (*ChangePassphraseResponse) ProtoMessage()               {}
func (*ChangePassphraseResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{26} }

type FundTransactionRequest struct {
	Account                  uint32 `protobuf:"varint,1,opt,name=account" json:"account,omitempty"`
	TargetAmount             int64  `protobuf:"varint,2,opt,name=target_amount,json=targetAmount" json:"target_amount,omitempty"`
	RequiredConfirmations    int32  `protobuf:"varint,3,opt,name=required_confirmations,json=requiredConfirmations" json:"required_confirmations,omitempty"`
	IncludeImmatureCoinbases bool   `protobuf:"varint,4,opt,name=include_immature_coinbases,json=includeImmatureCoinbases" json:"include_immature_coinbases,omitempty"`
	IncludeChangeScript      bool   `protobuf:"varint,5,opt,name=include_change_script,json=includeChangeScript" json:"include_change_script,omitempty"`
}

func (m *FundTransactionRequest) Reset()                    { *m = FundTransactionRequest{} }
func (m *FundTransactionRequest) String() string            { return proto.CompactTextString(m) }
func (*FundTransactionRequest) ProtoMessage()               {}
func (*FundTransactionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{27} }

func (m *FundTransactionRequest) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *FundTransactionRequest) GetTargetAmount() int64 {
	if m != nil {
		return m.TargetAmount
	}
	return 0
}

func (m *FundTransactionRequest) GetRequiredConfirmations() int32 {
	if m != nil {
		return m.RequiredConfirmations
	}
	return 0
}

func (m *FundTransactionRequest) GetIncludeImmatureCoinbases() bool {
	if m != nil {
		return m.IncludeImmatureCoinbases
	}
	return false
}

func (m *FundTransactionRequest) GetIncludeChangeScript() bool {
	if m != nil {
		return m.IncludeChangeScript
	}
	return false
}

type FundTransactionResponse struct {
	SelectedOutputs []*FundTransactionResponse_PreviousOutput `protobuf:"bytes,1,rep,name=selected_outputs,json=selectedOutputs" json:"selected_outputs,omitempty"`
	TotalAmount     int64                                     `protobuf:"varint,2,opt,name=total_amount,json=totalAmount" json:"total_amount,omitempty"`
	ChangePkScript  []byte                                    `protobuf:"bytes,3,opt,name=change_pk_script,json=changePkScript,proto3" json:"change_pk_script,omitempty"`
}

func (m *FundTransactionResponse) Reset()                    { *m = FundTransactionResponse{} }
func (m *FundTransactionResponse) String() string            { return proto.CompactTextString(m) }
func (*FundTransactionResponse) ProtoMessage()               {}
func (*FundTransactionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{28} }

func (m *FundTransactionResponse) GetSelectedOutputs() []*FundTransactionResponse_PreviousOutput {
	if m != nil {
		return m.SelectedOutputs
	}
	return nil
}

func (m *FundTransactionResponse) GetTotalAmount() int64 {
	if m != nil {
		return m.TotalAmount
	}
	return 0
}

func (m *FundTransactionResponse) GetChangePkScript() []byte {
	if m != nil {
		return m.ChangePkScript
	}
	return nil
}

type FundTransactionResponse_PreviousOutput struct {
	TransactionHash []byte `protobuf:"bytes,1,opt,name=transaction_hash,json=transactionHash,proto3" json:"transaction_hash,omitempty"`
	OutputIndex     uint32 `protobuf:"varint,2,opt,name=output_index,json=outputIndex" json:"output_index,omitempty"`
	Amount          int64  `protobuf:"varint,3,opt,name=amount" json:"amount,omitempty"`
	PkScript        []byte `protobuf:"bytes,4,opt,name=pk_script,json=pkScript,proto3" json:"pk_script,omitempty"`
	ReceiveTime     int64  `protobuf:"varint,5,opt,name=receive_time,json=receiveTime" json:"receive_time,omitempty"`
	FromCoinbase    bool   `protobuf:"varint,6,opt,name=from_coinbase,json=fromCoinbase" json:"from_coinbase,omitempty"`
}

func (m *FundTransactionResponse_PreviousOutput) Reset() {
	*m = FundTransactionResponse_PreviousOutput{}
}
func (m *FundTransactionResponse_PreviousOutput) String() string { return proto.CompactTextString(m) }
func (*FundTransactionResponse_PreviousOutput) ProtoMessage()    {}
func (*FundTransactionResponse_PreviousOutput) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{28, 0}
}

func (m *FundTransactionResponse_PreviousOutput) GetTransactionHash() []byte {
	if m != nil {
		return m.TransactionHash
	}
	return nil
}

func (m *FundTransactionResponse_PreviousOutput) GetOutputIndex() uint32 {
	if m != nil {
		return m.OutputIndex
	}
	return 0
}

func (m *FundTransactionResponse_PreviousOutput) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *FundTransactionResponse_PreviousOutput) GetPkScript() []byte {
	if m != nil {
		return m.PkScript
	}
	return nil
}

func (m *FundTransactionResponse_PreviousOutput) GetReceiveTime() int64 {
	if m != nil {
		return m.ReceiveTime
	}
	return 0
}

func (m *FundTransactionResponse_PreviousOutput) GetFromCoinbase() bool {
	if m != nil {
		return m.FromCoinbase
	}
	return false
}

type SignTransactionRequest struct {
	Passphrase            []byte `protobuf:"bytes,1,opt,name=passphrase,proto3" json:"passphrase,omitempty"`
	SerializedTransaction []byte `protobuf:"bytes,2,opt,name=serialized_transaction,json=serializedTransaction,proto3" json:"serialized_transaction,omitempty"`
//如果未指定索引，则将为添加签名脚本
//每一个输入。如果指定了任何输入索引，则只指定那些输入
//将被签署。而不是返回一个不完整的签名
//事务如果要签名的任何输入不能是
//立即出错。
	InputIndexes []uint32 `protobuf:"varint,3,rep,packed,name=input_indexes,json=inputIndexes" json:"input_indexes,omitempty"`
}

func (m *SignTransactionRequest) Reset()                    { *m = SignTransactionRequest{} }
func (m *SignTransactionRequest) String() string            { return proto.CompactTextString(m) }
func (*SignTransactionRequest) ProtoMessage()               {}
func (*SignTransactionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{29} }

func (m *SignTransactionRequest) GetPassphrase() []byte {
	if m != nil {
		return m.Passphrase
	}
	return nil
}

func (m *SignTransactionRequest) GetSerializedTransaction() []byte {
	if m != nil {
		return m.SerializedTransaction
	}
	return nil
}

func (m *SignTransactionRequest) GetInputIndexes() []uint32 {
	if m != nil {
		return m.InputIndexes
	}
	return nil
}

type SignTransactionResponse struct {
	Transaction          []byte   `protobuf:"bytes,1,opt,name=transaction,proto3" json:"transaction,omitempty"`
	UnsignedInputIndexes []uint32 `protobuf:"varint,2,rep,packed,name=unsigned_input_indexes,json=unsignedInputIndexes" json:"unsigned_input_indexes,omitempty"`
}

func (m *SignTransactionResponse) Reset()                    { *m = SignTransactionResponse{} }
func (m *SignTransactionResponse) String() string            { return proto.CompactTextString(m) }
func (*SignTransactionResponse) ProtoMessage()               {}
func (*SignTransactionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{30} }

func (m *SignTransactionResponse) GetTransaction() []byte {
	if m != nil {
		return m.Transaction
	}
	return nil
}

func (m *SignTransactionResponse) GetUnsignedInputIndexes() []uint32 {
	if m != nil {
		return m.UnsignedInputIndexes
	}
	return nil
}

type PublishTransactionRequest struct {
	SignedTransaction []byte `protobuf:"bytes,1,opt,name=signed_transaction,json=signedTransaction,proto3" json:"signed_transaction,omitempty"`
}

func (m *PublishTransactionRequest) Reset()                    { *m = PublishTransactionRequest{} }
func (m *PublishTransactionRequest) String() string            { return proto.CompactTextString(m) }
func (*PublishTransactionRequest) ProtoMessage()               {}
func (*PublishTransactionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{31} }

func (m *PublishTransactionRequest) GetSignedTransaction() []byte {
	if m != nil {
		return m.SignedTransaction
	}
	return nil
}

type PublishTransactionResponse struct {
}

func (m *PublishTransactionResponse) Reset()                    { *m = PublishTransactionResponse{} }
func (m *PublishTransactionResponse) String() string            { return proto.CompactTextString(m) }
func (*PublishTransactionResponse) ProtoMessage()               {}
func (*PublishTransactionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{32} }

type TransactionNotificationsRequest struct {
}

func (m *TransactionNotificationsRequest) Reset()         { *m = TransactionNotificationsRequest{} }
func (m *TransactionNotificationsRequest) String() string { return proto.CompactTextString(m) }
func (*TransactionNotificationsRequest) ProtoMessage()    {}
func (*TransactionNotificationsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{33}
}

type TransactionNotificationsResponse struct {
//按高度增加排序。这是一个重复的字段，有这么多新块
//
	AttachedBlocks []*BlockDetails `protobuf:"bytes,1,rep,name=attached_blocks,json=attachedBlocks" json:"attached_blocks,omitempty"`
//如果有连锁店重组，可能是钱包被堵了。
//不再处于最佳链中的事务。这些就是那些
//块的散列。
	DetachedBlocks [][]byte `protobuf:"bytes,2,rep,name=detached_blocks,json=detachedBlocks,proto3" json:"detached_blocks,omitempty"`
//这里包括所有新的未关联交易。这些非关联交易
//引用当前的最佳链，因此分离块的事务可能
//
//在新的链条中。另外，如果没有附加新块，但相关的
//钱包看到的是无限制交易，将在此处报告。
	UnminedTransactions []*TransactionDetails `protobuf:"bytes,3,rep,name=unmined_transactions,json=unminedTransactions" json:"unmined_transactions,omitempty"`
//而不是通知所有已删除的未关联交易，
//只需发送所有当前哈希。
	UnminedTransactionHashes [][]byte `protobuf:"bytes,4,rep,name=unmined_transaction_hashes,json=unminedTransactionHashes,proto3" json:"unmined_transaction_hashes,omitempty"`
}

func (m *TransactionNotificationsResponse) Reset()         { *m = TransactionNotificationsResponse{} }
func (m *TransactionNotificationsResponse) String() string { return proto.CompactTextString(m) }
func (*TransactionNotificationsResponse) ProtoMessage()    {}
func (*TransactionNotificationsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{34}
}

func (m *TransactionNotificationsResponse) GetAttachedBlocks() []*BlockDetails {
	if m != nil {
		return m.AttachedBlocks
	}
	return nil
}

func (m *TransactionNotificationsResponse) GetDetachedBlocks() [][]byte {
	if m != nil {
		return m.DetachedBlocks
	}
	return nil
}

func (m *TransactionNotificationsResponse) GetUnminedTransactions() []*TransactionDetails {
	if m != nil {
		return m.UnminedTransactions
	}
	return nil
}

func (m *TransactionNotificationsResponse) GetUnminedTransactionHashes() [][]byte {
	if m != nil {
		return m.UnminedTransactionHashes
	}
	return nil
}

type SpentnessNotificationsRequest struct {
	Account         uint32 `protobuf:"varint,1,opt,name=account" json:"account,omitempty"`
	NoNotifyUnspent bool   `protobuf:"varint,2,opt,name=no_notify_unspent,json=noNotifyUnspent" json:"no_notify_unspent,omitempty"`
	NoNotifySpent   bool   `protobuf:"varint,3,opt,name=no_notify_spent,json=noNotifySpent" json:"no_notify_spent,omitempty"`
}

func (m *SpentnessNotificationsRequest) Reset()                    { *m = SpentnessNotificationsRequest{} }
func (m *SpentnessNotificationsRequest) String() string            { return proto.CompactTextString(m) }
func (*SpentnessNotificationsRequest) ProtoMessage()               {}
func (*SpentnessNotificationsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{35} }

func (m *SpentnessNotificationsRequest) GetAccount() uint32 {
	if m != nil {
		return m.Account
	}
	return 0
}

func (m *SpentnessNotificationsRequest) GetNoNotifyUnspent() bool {
	if m != nil {
		return m.NoNotifyUnspent
	}
	return false
}

func (m *SpentnessNotificationsRequest) GetNoNotifySpent() bool {
	if m != nil {
		return m.NoNotifySpent
	}
	return false
}

type SpentnessNotificationsResponse struct {
	TransactionHash []byte                                  `protobuf:"bytes,1,opt,name=transaction_hash,json=transactionHash,proto3" json:"transaction_hash,omitempty"`
	OutputIndex     uint32                                  `protobuf:"varint,2,opt,name=output_index,json=outputIndex" json:"output_index,omitempty"`
	Spender         *SpentnessNotificationsResponse_Spender `protobuf:"bytes,3,opt,name=spender" json:"spender,omitempty"`
}

func (m *SpentnessNotificationsResponse) Reset()                    { *m = SpentnessNotificationsResponse{} }
func (m *SpentnessNotificationsResponse) String() string            { return proto.CompactTextString(m) }
func (*SpentnessNotificationsResponse) ProtoMessage()               {}
func (*SpentnessNotificationsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{36} }

func (m *SpentnessNotificationsResponse) GetTransactionHash() []byte {
	if m != nil {
		return m.TransactionHash
	}
	return nil
}

func (m *SpentnessNotificationsResponse) GetOutputIndex() uint32 {
	if m != nil {
		return m.OutputIndex
	}
	return 0
}

func (m *SpentnessNotificationsResponse) GetSpender() *SpentnessNotificationsResponse_Spender {
	if m != nil {
		return m.Spender
	}
	return nil
}

type SpentnessNotificationsResponse_Spender struct {
	TransactionHash []byte `protobuf:"bytes,1,opt,name=transaction_hash,json=transactionHash,proto3" json:"transaction_hash,omitempty"`
	InputIndex      uint32 `protobuf:"varint,2,opt,name=input_index,json=inputIndex" json:"input_index,omitempty"`
}

func (m *SpentnessNotificationsResponse_Spender) Reset() {
	*m = SpentnessNotificationsResponse_Spender{}
}
func (m *SpentnessNotificationsResponse_Spender) String() string { return proto.CompactTextString(m) }
func (*SpentnessNotificationsResponse_Spender) ProtoMessage()    {}
func (*SpentnessNotificationsResponse_Spender) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{36, 0}
}

func (m *SpentnessNotificationsResponse_Spender) GetTransactionHash() []byte {
	if m != nil {
		return m.TransactionHash
	}
	return nil
}

func (m *SpentnessNotificationsResponse_Spender) GetInputIndex() uint32 {
	if m != nil {
		return m.InputIndex
	}
	return 0
}

type AccountNotificationsRequest struct {
}

func (m *AccountNotificationsRequest) Reset()                    { *m = AccountNotificationsRequest{} }
func (m *AccountNotificationsRequest) String() string            { return proto.CompactTextString(m) }
func (*AccountNotificationsRequest) ProtoMessage()               {}
func (*AccountNotificationsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{37} }

type AccountNotificationsResponse struct {
	AccountNumber    uint32 `protobuf:"varint,1,opt,name=account_number,json=accountNumber" json:"account_number,omitempty"`
	AccountName      string `protobuf:"bytes,2,opt,name=account_name,json=accountName" json:"account_name,omitempty"`
	ExternalKeyCount uint32 `protobuf:"varint,3,opt,name=external_key_count,json=externalKeyCount" json:"external_key_count,omitempty"`
	InternalKeyCount uint32 `protobuf:"varint,4,opt,name=internal_key_count,json=internalKeyCount" json:"internal_key_count,omitempty"`
	ImportedKeyCount uint32 `protobuf:"varint,5,opt,name=imported_key_count,json=importedKeyCount" json:"imported_key_count,omitempty"`
}

func (m *AccountNotificationsResponse) Reset()                    { *m = AccountNotificationsResponse{} }
func (m *AccountNotificationsResponse) String() string            { return proto.CompactTextString(m) }
func (*AccountNotificationsResponse) ProtoMessage()               {}
func (*AccountNotificationsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{38} }

func (m *AccountNotificationsResponse) GetAccountNumber() uint32 {
	if m != nil {
		return m.AccountNumber
	}
	return 0
}

func (m *AccountNotificationsResponse) GetAccountName() string {
	if m != nil {
		return m.AccountName
	}
	return ""
}

func (m *AccountNotificationsResponse) GetExternalKeyCount() uint32 {
	if m != nil {
		return m.ExternalKeyCount
	}
	return 0
}

func (m *AccountNotificationsResponse) GetInternalKeyCount() uint32 {
	if m != nil {
		return m.InternalKeyCount
	}
	return 0
}

func (m *AccountNotificationsResponse) GetImportedKeyCount() uint32 {
	if m != nil {
		return m.ImportedKeyCount
	}
	return 0
}

type CreateWalletRequest struct {
	PublicPassphrase  []byte `protobuf:"bytes,1,opt,name=public_passphrase,json=publicPassphrase,proto3" json:"public_passphrase,omitempty"`
	PrivatePassphrase []byte `protobuf:"bytes,2,opt,name=private_passphrase,json=privatePassphrase,proto3" json:"private_passphrase,omitempty"`
	Seed              []byte `protobuf:"bytes,3,opt,name=seed,proto3" json:"seed,omitempty"`
}

func (m *CreateWalletRequest) Reset()                    { *m = CreateWalletRequest{} }
func (m *CreateWalletRequest) String() string            { return proto.CompactTextString(m) }
func (*CreateWalletRequest) ProtoMessage()               {}
func (*CreateWalletRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{39} }

func (m *CreateWalletRequest) GetPublicPassphrase() []byte {
	if m != nil {
		return m.PublicPassphrase
	}
	return nil
}

func (m *CreateWalletRequest) GetPrivatePassphrase() []byte {
	if m != nil {
		return m.PrivatePassphrase
	}
	return nil
}

func (m *CreateWalletRequest) GetSeed() []byte {
	if m != nil {
		return m.Seed
	}
	return nil
}

type CreateWalletResponse struct {
}

func (m *CreateWalletResponse) Reset()                    { *m = CreateWalletResponse{} }
func (m *CreateWalletResponse) String() string            { return proto.CompactTextString(m) }
func (*CreateWalletResponse) ProtoMessage()               {}
func (*CreateWalletResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{40} }

type OpenWalletRequest struct {
	PublicPassphrase []byte `protobuf:"bytes,1,opt,name=public_passphrase,json=publicPassphrase,proto3" json:"public_passphrase,omitempty"`
}

func (m *OpenWalletRequest) Reset()                    { *m = OpenWalletRequest{} }
func (m *OpenWalletRequest) String() string            { return proto.CompactTextString(m) }
func (*OpenWalletRequest) ProtoMessage()               {}
func (*OpenWalletRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{41} }

func (m *OpenWalletRequest) GetPublicPassphrase() []byte {
	if m != nil {
		return m.PublicPassphrase
	}
	return nil
}

type OpenWalletResponse struct {
}

func (m *OpenWalletResponse) Reset()                    { *m = OpenWalletResponse{} }
func (m *OpenWalletResponse) String() string            { return proto.CompactTextString(m) }
func (*OpenWalletResponse) ProtoMessage()               {}
func (*OpenWalletResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{42} }

type CloseWalletRequest struct {
}

func (m *CloseWalletRequest) Reset()                    { *m = CloseWalletRequest{} }
func (m *CloseWalletRequest) String() string            { return proto.CompactTextString(m) }
func (*CloseWalletRequest) ProtoMessage()               {}
func (*CloseWalletRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{43} }

type CloseWalletResponse struct {
}

func (m *CloseWalletResponse) Reset()                    { *m = CloseWalletResponse{} }
func (m *CloseWalletResponse) String() string            { return proto.CompactTextString(m) }
func (*CloseWalletResponse) ProtoMessage()               {}
func (*CloseWalletResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{44} }

type WalletExistsRequest struct {
}

func (m *WalletExistsRequest) Reset()                    { *m = WalletExistsRequest{} }
func (m *WalletExistsRequest) String() string            { return proto.CompactTextString(m) }
func (*WalletExistsRequest) ProtoMessage()               {}
func (*WalletExistsRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{45} }

type WalletExistsResponse struct {
	Exists bool `protobuf:"varint,1,opt,name=exists" json:"exists,omitempty"`
}

func (m *WalletExistsResponse) Reset()                    { *m = WalletExistsResponse{} }
func (m *WalletExistsResponse) String() string            { return proto.CompactTextString(m) }
func (*WalletExistsResponse) ProtoMessage()               {}
func (*WalletExistsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{46} }

func (m *WalletExistsResponse) GetExists() bool {
	if m != nil {
		return m.Exists
	}
	return false
}

type StartConsensusRpcRequest struct {
	NetworkAddress string `protobuf:"bytes,1,opt,name=network_address,json=networkAddress" json:"network_address,omitempty"`
	Username       string `protobuf:"bytes,2,opt,name=username" json:"username,omitempty"`
	Password       []byte `protobuf:"bytes,3,opt,name=password,proto3" json:"password,omitempty"`
	Certificate    []byte `protobuf:"bytes,4,opt,name=certificate,proto3" json:"certificate,omitempty"`
}

func (m *StartConsensusRpcRequest) Reset()                    { *m = StartConsensusRpcRequest{} }
func (m *StartConsensusRpcRequest) String() string            { return proto.CompactTextString(m) }
func (*StartConsensusRpcRequest) ProtoMessage()               {}
func (*StartConsensusRpcRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{47} }

func (m *StartConsensusRpcRequest) GetNetworkAddress() string {
	if m != nil {
		return m.NetworkAddress
	}
	return ""
}

func (m *StartConsensusRpcRequest) GetUsername() string {
	if m != nil {
		return m.Username
	}
	return ""
}

func (m *StartConsensusRpcRequest) GetPassword() []byte {
	if m != nil {
		return m.Password
	}
	return nil
}

func (m *StartConsensusRpcRequest) GetCertificate() []byte {
	if m != nil {
		return m.Certificate
	}
	return nil
}

type StartConsensusRpcResponse struct {
}

func (m *StartConsensusRpcResponse) Reset()                    { *m = StartConsensusRpcResponse{} }
func (m *StartConsensusRpcResponse) String() string            { return proto.CompactTextString(m) }
func (*StartConsensusRpcResponse) ProtoMessage()               {}
func (*StartConsensusRpcResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{48} }

func init() {
	proto.RegisterType((*VersionRequest)(nil), "walletrpc.VersionRequest")
	proto.RegisterType((*VersionResponse)(nil), "walletrpc.VersionResponse")
	proto.RegisterType((*TransactionDetails)(nil), "walletrpc.TransactionDetails")
	proto.RegisterType((*TransactionDetails_Input)(nil), "walletrpc.TransactionDetails.Input")
	proto.RegisterType((*TransactionDetails_Output)(nil), "walletrpc.TransactionDetails.Output")
	proto.RegisterType((*BlockDetails)(nil), "walletrpc.BlockDetails")
	proto.RegisterType((*AccountBalance)(nil), "walletrpc.AccountBalance")
	proto.RegisterType((*PingRequest)(nil), "walletrpc.PingRequest")
	proto.RegisterType((*PingResponse)(nil), "walletrpc.PingResponse")
	proto.RegisterType((*NetworkRequest)(nil), "walletrpc.NetworkRequest")
	proto.RegisterType((*NetworkResponse)(nil), "walletrpc.NetworkResponse")
	proto.RegisterType((*AccountNumberRequest)(nil), "walletrpc.AccountNumberRequest")
	proto.RegisterType((*AccountNumberResponse)(nil), "walletrpc.AccountNumberResponse")
	proto.RegisterType((*AccountsRequest)(nil), "walletrpc.AccountsRequest")
	proto.RegisterType((*AccountsResponse)(nil), "walletrpc.AccountsResponse")
	proto.RegisterType((*AccountsResponse_Account)(nil), "walletrpc.AccountsResponse.Account")
	proto.RegisterType((*RenameAccountRequest)(nil), "walletrpc.RenameAccountRequest")
	proto.RegisterType((*RenameAccountResponse)(nil), "walletrpc.RenameAccountResponse")
	proto.RegisterType((*NextAccountRequest)(nil), "walletrpc.NextAccountRequest")
	proto.RegisterType((*NextAccountResponse)(nil), "walletrpc.NextAccountResponse")
	proto.RegisterType((*NextAddressRequest)(nil), "walletrpc.NextAddressRequest")
	proto.RegisterType((*NextAddressResponse)(nil), "walletrpc.NextAddressResponse")
	proto.RegisterType((*ImportPrivateKeyRequest)(nil), "walletrpc.ImportPrivateKeyRequest")
	proto.RegisterType((*ImportPrivateKeyResponse)(nil), "walletrpc.ImportPrivateKeyResponse")
	proto.RegisterType((*BalanceRequest)(nil), "walletrpc.BalanceRequest")
	proto.RegisterType((*BalanceResponse)(nil), "walletrpc.BalanceResponse")
	proto.RegisterType((*GetTransactionsRequest)(nil), "walletrpc.GetTransactionsRequest")
	proto.RegisterType((*GetTransactionsResponse)(nil), "walletrpc.GetTransactionsResponse")
	proto.RegisterType((*ChangePassphraseRequest)(nil), "walletrpc.ChangePassphraseRequest")
	proto.RegisterType((*ChangePassphraseResponse)(nil), "walletrpc.ChangePassphraseResponse")
	proto.RegisterType((*FundTransactionRequest)(nil), "walletrpc.FundTransactionRequest")
	proto.RegisterType((*FundTransactionResponse)(nil), "walletrpc.FundTransactionResponse")
	proto.RegisterType((*FundTransactionResponse_PreviousOutput)(nil), "walletrpc.FundTransactionResponse.PreviousOutput")
	proto.RegisterType((*SignTransactionRequest)(nil), "walletrpc.SignTransactionRequest")
	proto.RegisterType((*SignTransactionResponse)(nil), "walletrpc.SignTransactionResponse")
	proto.RegisterType((*PublishTransactionRequest)(nil), "walletrpc.PublishTransactionRequest")
	proto.RegisterType((*PublishTransactionResponse)(nil), "walletrpc.PublishTransactionResponse")
	proto.RegisterType((*TransactionNotificationsRequest)(nil), "walletrpc.TransactionNotificationsRequest")
	proto.RegisterType((*TransactionNotificationsResponse)(nil), "walletrpc.TransactionNotificationsResponse")
	proto.RegisterType((*SpentnessNotificationsRequest)(nil), "walletrpc.SpentnessNotificationsRequest")
	proto.RegisterType((*SpentnessNotificationsResponse)(nil), "walletrpc.SpentnessNotificationsResponse")
	proto.RegisterType((*SpentnessNotificationsResponse_Spender)(nil), "walletrpc.SpentnessNotificationsResponse.Spender")
	proto.RegisterType((*AccountNotificationsRequest)(nil), "walletrpc.AccountNotificationsRequest")
	proto.RegisterType((*AccountNotificationsResponse)(nil), "walletrpc.AccountNotificationsResponse")
	proto.RegisterType((*CreateWalletRequest)(nil), "walletrpc.CreateWalletRequest")
	proto.RegisterType((*CreateWalletResponse)(nil), "walletrpc.CreateWalletResponse")
	proto.RegisterType((*OpenWalletRequest)(nil), "walletrpc.OpenWalletRequest")
	proto.RegisterType((*OpenWalletResponse)(nil), "walletrpc.OpenWalletResponse")
	proto.RegisterType((*CloseWalletRequest)(nil), "walletrpc.CloseWalletRequest")
	proto.RegisterType((*CloseWalletResponse)(nil), "walletrpc.CloseWalletResponse")
	proto.RegisterType((*WalletExistsRequest)(nil), "walletrpc.WalletExistsRequest")
	proto.RegisterType((*WalletExistsResponse)(nil), "walletrpc.WalletExistsResponse")
	proto.RegisterType((*StartConsensusRpcRequest)(nil), "walletrpc.StartConsensusRpcRequest")
	proto.RegisterType((*StartConsensusRpcResponse)(nil), "walletrpc.StartConsensusRpcResponse")
	proto.RegisterEnum("walletrpc.NextAddressRequest_Kind", NextAddressRequest_Kind_name, NextAddressRequest_Kind_value)
	proto.RegisterEnum("walletrpc.ChangePassphraseRequest_Key", ChangePassphraseRequest_Key_name, ChangePassphraseRequest_Key_value)
}

//引用导入以禁止错误（如果未使用）。
var _ context.Context
var _ grpc.ClientConn

//这是一个编译时断言，以确保生成的文件
//与正在编译的GRPC包兼容。
const _ = grpc.SupportPackageIsVersion4

//VersionService服务的客户端API

type VersionServiceClient interface {
	Version(ctx context.Context, in *VersionRequest, opts ...grpc.CallOption) (*VersionResponse, error)
}

type versionServiceClient struct {
	cc *grpc.ClientConn
}

func NewVersionServiceClient(cc *grpc.ClientConn) VersionServiceClient {
	return &versionServiceClient{cc}
}

func (c *versionServiceClient) Version(ctx context.Context, in *VersionRequest, opts ...grpc.CallOption) (*VersionResponse, error) {
	out := new(VersionResponse)
	err := grpc.Invoke(ctx, "/walletrpc.VersionService/Version", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//VersionService服务的服务器API

type VersionServiceServer interface {
	Version(context.Context, *VersionRequest) (*VersionResponse, error)
}

func RegisterVersionServiceServer(s *grpc.Server, srv VersionServiceServer) {
	s.RegisterService(&_VersionService_serviceDesc, srv)
}

func _VersionService_Version_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VersionServiceServer).Version(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.VersionService/Version",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VersionServiceServer).Version(ctx, req.(*VersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _VersionService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "walletrpc.VersionService",
	HandlerType: (*VersionServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Version",
			Handler:    _VersionService_Version_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}

//用于WalletService服务的客户端API

type WalletServiceClient interface {
//查询
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	Network(ctx context.Context, in *NetworkRequest, opts ...grpc.CallOption) (*NetworkResponse, error)
	AccountNumber(ctx context.Context, in *AccountNumberRequest, opts ...grpc.CallOption) (*AccountNumberResponse, error)
	Accounts(ctx context.Context, in *AccountsRequest, opts ...grpc.CallOption) (*AccountsResponse, error)
	Balance(ctx context.Context, in *BalanceRequest, opts ...grpc.CallOption) (*BalanceResponse, error)
	GetTransactions(ctx context.Context, in *GetTransactionsRequest, opts ...grpc.CallOption) (*GetTransactionsResponse, error)
//通知
	TransactionNotifications(ctx context.Context, in *TransactionNotificationsRequest, opts ...grpc.CallOption) (WalletService_TransactionNotificationsClient, error)
	SpentnessNotifications(ctx context.Context, in *SpentnessNotificationsRequest, opts ...grpc.CallOption) (WalletService_SpentnessNotificationsClient, error)
	AccountNotifications(ctx context.Context, in *AccountNotificationsRequest, opts ...grpc.CallOption) (WalletService_AccountNotificationsClient, error)
//控制
	ChangePassphrase(ctx context.Context, in *ChangePassphraseRequest, opts ...grpc.CallOption) (*ChangePassphraseResponse, error)
	RenameAccount(ctx context.Context, in *RenameAccountRequest, opts ...grpc.CallOption) (*RenameAccountResponse, error)
	NextAccount(ctx context.Context, in *NextAccountRequest, opts ...grpc.CallOption) (*NextAccountResponse, error)
	NextAddress(ctx context.Context, in *NextAddressRequest, opts ...grpc.CallOption) (*NextAddressResponse, error)
	ImportPrivateKey(ctx context.Context, in *ImportPrivateKeyRequest, opts ...grpc.CallOption) (*ImportPrivateKeyResponse, error)
	FundTransaction(ctx context.Context, in *FundTransactionRequest, opts ...grpc.CallOption) (*FundTransactionResponse, error)
	SignTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error)
	PublishTransaction(ctx context.Context, in *PublishTransactionRequest, opts ...grpc.CallOption) (*PublishTransactionResponse, error)
}

type walletServiceClient struct {
	cc *grpc.ClientConn
}

func NewWalletServiceClient(cc *grpc.ClientConn) WalletServiceClient {
	return &walletServiceClient{cc}
}

func (c *walletServiceClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/Ping", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) Network(ctx context.Context, in *NetworkRequest, opts ...grpc.CallOption) (*NetworkResponse, error) {
	out := new(NetworkResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/Network", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) AccountNumber(ctx context.Context, in *AccountNumberRequest, opts ...grpc.CallOption) (*AccountNumberResponse, error) {
	out := new(AccountNumberResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/AccountNumber", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) Accounts(ctx context.Context, in *AccountsRequest, opts ...grpc.CallOption) (*AccountsResponse, error) {
	out := new(AccountsResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/Accounts", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) Balance(ctx context.Context, in *BalanceRequest, opts ...grpc.CallOption) (*BalanceResponse, error) {
	out := new(BalanceResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/Balance", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) GetTransactions(ctx context.Context, in *GetTransactionsRequest, opts ...grpc.CallOption) (*GetTransactionsResponse, error) {
	out := new(GetTransactionsResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/GetTransactions", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) TransactionNotifications(ctx context.Context, in *TransactionNotificationsRequest, opts ...grpc.CallOption) (WalletService_TransactionNotificationsClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_WalletService_serviceDesc.Streams[0], c.cc, "/walletrpc.WalletService/TransactionNotifications", opts...)
	if err != nil {
		return nil, err
	}
	x := &walletServiceTransactionNotificationsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WalletService_TransactionNotificationsClient interface {
	Recv() (*TransactionNotificationsResponse, error)
	grpc.ClientStream
}

type walletServiceTransactionNotificationsClient struct {
	grpc.ClientStream
}

func (x *walletServiceTransactionNotificationsClient) Recv() (*TransactionNotificationsResponse, error) {
	m := new(TransactionNotificationsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *walletServiceClient) SpentnessNotifications(ctx context.Context, in *SpentnessNotificationsRequest, opts ...grpc.CallOption) (WalletService_SpentnessNotificationsClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_WalletService_serviceDesc.Streams[1], c.cc, "/walletrpc.WalletService/SpentnessNotifications", opts...)
	if err != nil {
		return nil, err
	}
	x := &walletServiceSpentnessNotificationsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WalletService_SpentnessNotificationsClient interface {
	Recv() (*SpentnessNotificationsResponse, error)
	grpc.ClientStream
}

type walletServiceSpentnessNotificationsClient struct {
	grpc.ClientStream
}

func (x *walletServiceSpentnessNotificationsClient) Recv() (*SpentnessNotificationsResponse, error) {
	m := new(SpentnessNotificationsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *walletServiceClient) AccountNotifications(ctx context.Context, in *AccountNotificationsRequest, opts ...grpc.CallOption) (WalletService_AccountNotificationsClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_WalletService_serviceDesc.Streams[2], c.cc, "/walletrpc.WalletService/AccountNotifications", opts...)
	if err != nil {
		return nil, err
	}
	x := &walletServiceAccountNotificationsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WalletService_AccountNotificationsClient interface {
	Recv() (*AccountNotificationsResponse, error)
	grpc.ClientStream
}

type walletServiceAccountNotificationsClient struct {
	grpc.ClientStream
}

func (x *walletServiceAccountNotificationsClient) Recv() (*AccountNotificationsResponse, error) {
	m := new(AccountNotificationsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *walletServiceClient) ChangePassphrase(ctx context.Context, in *ChangePassphraseRequest, opts ...grpc.CallOption) (*ChangePassphraseResponse, error) {
	out := new(ChangePassphraseResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/ChangePassphrase", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) RenameAccount(ctx context.Context, in *RenameAccountRequest, opts ...grpc.CallOption) (*RenameAccountResponse, error) {
	out := new(RenameAccountResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/RenameAccount", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) NextAccount(ctx context.Context, in *NextAccountRequest, opts ...grpc.CallOption) (*NextAccountResponse, error) {
	out := new(NextAccountResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/NextAccount", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) NextAddress(ctx context.Context, in *NextAddressRequest, opts ...grpc.CallOption) (*NextAddressResponse, error) {
	out := new(NextAddressResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/NextAddress", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) ImportPrivateKey(ctx context.Context, in *ImportPrivateKeyRequest, opts ...grpc.CallOption) (*ImportPrivateKeyResponse, error) {
	out := new(ImportPrivateKeyResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/ImportPrivateKey", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) FundTransaction(ctx context.Context, in *FundTransactionRequest, opts ...grpc.CallOption) (*FundTransactionResponse, error) {
	out := new(FundTransactionResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/FundTransaction", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) SignTransaction(ctx context.Context, in *SignTransactionRequest, opts ...grpc.CallOption) (*SignTransactionResponse, error) {
	out := new(SignTransactionResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/SignTransaction", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletServiceClient) PublishTransaction(ctx context.Context, in *PublishTransactionRequest, opts ...grpc.CallOption) (*PublishTransactionResponse, error) {
	out := new(PublishTransactionResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletService/PublishTransaction", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//用于WalletService服务的服务器API

type WalletServiceServer interface {
//查询
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	Network(context.Context, *NetworkRequest) (*NetworkResponse, error)
	AccountNumber(context.Context, *AccountNumberRequest) (*AccountNumberResponse, error)
	Accounts(context.Context, *AccountsRequest) (*AccountsResponse, error)
	Balance(context.Context, *BalanceRequest) (*BalanceResponse, error)
	GetTransactions(context.Context, *GetTransactionsRequest) (*GetTransactionsResponse, error)
//通知
	TransactionNotifications(*TransactionNotificationsRequest, WalletService_TransactionNotificationsServer) error
	SpentnessNotifications(*SpentnessNotificationsRequest, WalletService_SpentnessNotificationsServer) error
	AccountNotifications(*AccountNotificationsRequest, WalletService_AccountNotificationsServer) error
//控制
	ChangePassphrase(context.Context, *ChangePassphraseRequest) (*ChangePassphraseResponse, error)
	RenameAccount(context.Context, *RenameAccountRequest) (*RenameAccountResponse, error)
	NextAccount(context.Context, *NextAccountRequest) (*NextAccountResponse, error)
	NextAddress(context.Context, *NextAddressRequest) (*NextAddressResponse, error)
	ImportPrivateKey(context.Context, *ImportPrivateKeyRequest) (*ImportPrivateKeyResponse, error)
	FundTransaction(context.Context, *FundTransactionRequest) (*FundTransactionResponse, error)
	SignTransaction(context.Context, *SignTransactionRequest) (*SignTransactionResponse, error)
	PublishTransaction(context.Context, *PublishTransactionRequest) (*PublishTransactionResponse, error)
}

func RegisterWalletServiceServer(s *grpc.Server, srv WalletServiceServer) {
	s.RegisterService(&_WalletService_serviceDesc, srv)
}

func _WalletService_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_Network_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NetworkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).Network(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/Network",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).Network(ctx, req.(*NetworkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_AccountNumber_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountNumberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).AccountNumber(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/AccountNumber",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).AccountNumber(ctx, req.(*AccountNumberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_Accounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).Accounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/Accounts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).Accounts(ctx, req.(*AccountsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_Balance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).Balance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/Balance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).Balance(ctx, req.(*BalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_GetTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTransactionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).GetTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/GetTransactions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).GetTransactions(ctx, req.(*GetTransactionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_TransactionNotifications_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(TransactionNotificationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WalletServiceServer).TransactionNotifications(m, &walletServiceTransactionNotificationsServer{stream})
}

type WalletService_TransactionNotificationsServer interface {
	Send(*TransactionNotificationsResponse) error
	grpc.ServerStream
}

type walletServiceTransactionNotificationsServer struct {
	grpc.ServerStream
}

func (x *walletServiceTransactionNotificationsServer) Send(m *TransactionNotificationsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _WalletService_SpentnessNotifications_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SpentnessNotificationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WalletServiceServer).SpentnessNotifications(m, &walletServiceSpentnessNotificationsServer{stream})
}

type WalletService_SpentnessNotificationsServer interface {
	Send(*SpentnessNotificationsResponse) error
	grpc.ServerStream
}

type walletServiceSpentnessNotificationsServer struct {
	grpc.ServerStream
}

func (x *walletServiceSpentnessNotificationsServer) Send(m *SpentnessNotificationsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _WalletService_AccountNotifications_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(AccountNotificationsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WalletServiceServer).AccountNotifications(m, &walletServiceAccountNotificationsServer{stream})
}

type WalletService_AccountNotificationsServer interface {
	Send(*AccountNotificationsResponse) error
	grpc.ServerStream
}

type walletServiceAccountNotificationsServer struct {
	grpc.ServerStream
}

func (x *walletServiceAccountNotificationsServer) Send(m *AccountNotificationsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _WalletService_ChangePassphrase_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ChangePassphraseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).ChangePassphrase(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/ChangePassphrase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).ChangePassphrase(ctx, req.(*ChangePassphraseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_RenameAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RenameAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).RenameAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/RenameAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).RenameAccount(ctx, req.(*RenameAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_NextAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NextAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).NextAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/NextAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).NextAccount(ctx, req.(*NextAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_NextAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NextAddressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).NextAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/NextAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).NextAddress(ctx, req.(*NextAddressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_ImportPrivateKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ImportPrivateKeyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).ImportPrivateKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/ImportPrivateKey",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).ImportPrivateKey(ctx, req.(*ImportPrivateKeyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_FundTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FundTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).FundTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/FundTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).FundTransaction(ctx, req.(*FundTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_SignTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).SignTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/SignTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).SignTransaction(ctx, req.(*SignTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletService_PublishTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PublishTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletServiceServer).PublishTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletService/PublishTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletServiceServer).PublishTransaction(ctx, req.(*PublishTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _WalletService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "walletrpc.WalletService",
	HandlerType: (*WalletServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _WalletService_Ping_Handler,
		},
		{
			MethodName: "Network",
			Handler:    _WalletService_Network_Handler,
		},
		{
			MethodName: "AccountNumber",
			Handler:    _WalletService_AccountNumber_Handler,
		},
		{
			MethodName: "Accounts",
			Handler:    _WalletService_Accounts_Handler,
		},
		{
			MethodName: "Balance",
			Handler:    _WalletService_Balance_Handler,
		},
		{
			MethodName: "GetTransactions",
			Handler:    _WalletService_GetTransactions_Handler,
		},
		{
			MethodName: "ChangePassphrase",
			Handler:    _WalletService_ChangePassphrase_Handler,
		},
		{
			MethodName: "RenameAccount",
			Handler:    _WalletService_RenameAccount_Handler,
		},
		{
			MethodName: "NextAccount",
			Handler:    _WalletService_NextAccount_Handler,
		},
		{
			MethodName: "NextAddress",
			Handler:    _WalletService_NextAddress_Handler,
		},
		{
			MethodName: "ImportPrivateKey",
			Handler:    _WalletService_ImportPrivateKey_Handler,
		},
		{
			MethodName: "FundTransaction",
			Handler:    _WalletService_FundTransaction_Handler,
		},
		{
			MethodName: "SignTransaction",
			Handler:    _WalletService_SignTransaction_Handler,
		},
		{
			MethodName: "PublishTransaction",
			Handler:    _WalletService_PublishTransaction_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "TransactionNotifications",
			Handler:       _WalletService_TransactionNotifications_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SpentnessNotifications",
			Handler:       _WalletService_SpentnessNotifications_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "AccountNotifications",
			Handler:       _WalletService_AccountNotifications_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api.proto",
}

//用于WalletLoaderService服务的客户端API

type WalletLoaderServiceClient interface {
	WalletExists(ctx context.Context, in *WalletExistsRequest, opts ...grpc.CallOption) (*WalletExistsResponse, error)
	CreateWallet(ctx context.Context, in *CreateWalletRequest, opts ...grpc.CallOption) (*CreateWalletResponse, error)
	OpenWallet(ctx context.Context, in *OpenWalletRequest, opts ...grpc.CallOption) (*OpenWalletResponse, error)
	CloseWallet(ctx context.Context, in *CloseWalletRequest, opts ...grpc.CallOption) (*CloseWalletResponse, error)
	StartConsensusRpc(ctx context.Context, in *StartConsensusRpcRequest, opts ...grpc.CallOption) (*StartConsensusRpcResponse, error)
}

type walletLoaderServiceClient struct {
	cc *grpc.ClientConn
}

func NewWalletLoaderServiceClient(cc *grpc.ClientConn) WalletLoaderServiceClient {
	return &walletLoaderServiceClient{cc}
}

func (c *walletLoaderServiceClient) WalletExists(ctx context.Context, in *WalletExistsRequest, opts ...grpc.CallOption) (*WalletExistsResponse, error) {
	out := new(WalletExistsResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletLoaderService/WalletExists", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletLoaderServiceClient) CreateWallet(ctx context.Context, in *CreateWalletRequest, opts ...grpc.CallOption) (*CreateWalletResponse, error) {
	out := new(CreateWalletResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletLoaderService/CreateWallet", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletLoaderServiceClient) OpenWallet(ctx context.Context, in *OpenWalletRequest, opts ...grpc.CallOption) (*OpenWalletResponse, error) {
	out := new(OpenWalletResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletLoaderService/OpenWallet", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletLoaderServiceClient) CloseWallet(ctx context.Context, in *CloseWalletRequest, opts ...grpc.CallOption) (*CloseWalletResponse, error) {
	out := new(CloseWalletResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletLoaderService/CloseWallet", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *walletLoaderServiceClient) StartConsensusRpc(ctx context.Context, in *StartConsensusRpcRequest, opts ...grpc.CallOption) (*StartConsensusRpcResponse, error) {
	out := new(StartConsensusRpcResponse)
	err := grpc.Invoke(ctx, "/walletrpc.WalletLoaderService/StartConsensusRpc", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//用于WalletLoaderService服务的服务器API

type WalletLoaderServiceServer interface {
	WalletExists(context.Context, *WalletExistsRequest) (*WalletExistsResponse, error)
	CreateWallet(context.Context, *CreateWalletRequest) (*CreateWalletResponse, error)
	OpenWallet(context.Context, *OpenWalletRequest) (*OpenWalletResponse, error)
	CloseWallet(context.Context, *CloseWalletRequest) (*CloseWalletResponse, error)
	StartConsensusRpc(context.Context, *StartConsensusRpcRequest) (*StartConsensusRpcResponse, error)
}

func RegisterWalletLoaderServiceServer(s *grpc.Server, srv WalletLoaderServiceServer) {
	s.RegisterService(&_WalletLoaderService_serviceDesc, srv)
}

func _WalletLoaderService_WalletExists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WalletExistsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletLoaderServiceServer).WalletExists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletLoaderService/WalletExists",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletLoaderServiceServer).WalletExists(ctx, req.(*WalletExistsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletLoaderService_CreateWallet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateWalletRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletLoaderServiceServer).CreateWallet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletLoaderService/CreateWallet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletLoaderServiceServer).CreateWallet(ctx, req.(*CreateWalletRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletLoaderService_OpenWallet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OpenWalletRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletLoaderServiceServer).OpenWallet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletLoaderService/OpenWallet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletLoaderServiceServer).OpenWallet(ctx, req.(*OpenWalletRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletLoaderService_CloseWallet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CloseWalletRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletLoaderServiceServer).CloseWallet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletLoaderService/CloseWallet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletLoaderServiceServer).CloseWallet(ctx, req.(*CloseWalletRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WalletLoaderService_StartConsensusRpc_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartConsensusRpcRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WalletLoaderServiceServer).StartConsensusRpc(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/walletrpc.WalletLoaderService/StartConsensusRpc",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WalletLoaderServiceServer).StartConsensusRpc(ctx, req.(*StartConsensusRpcRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _WalletLoaderService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "walletrpc.WalletLoaderService",
	HandlerType: (*WalletLoaderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "WalletExists",
			Handler:    _WalletLoaderService_WalletExists_Handler,
		},
		{
			MethodName: "CreateWallet",
			Handler:    _WalletLoaderService_CreateWallet_Handler,
		},
		{
			MethodName: "OpenWallet",
			Handler:    _WalletLoaderService_OpenWallet_Handler,
		},
		{
			MethodName: "CloseWallet",
			Handler:    _WalletLoaderService_CloseWallet_Handler,
		},
		{
			MethodName: "StartConsensusRpc",
			Handler:    _WalletLoaderService_StartConsensusRpc_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}

func init() { proto.RegisterFile("api.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
//gzip文件描述符或协议的2391字节
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xac, 0x59, 0x5b, 0x73, 0xdc, 0x48,
	0x15, 0x66, 0x2c, 0xdf, 0x72, 0xe6, 0xde, 0xbe, 0x4d, 0x94, 0x38, 0xc9, 0x2a, 0xbb, 0xd9, 0xec,
	0x2e, 0x98, 0x60, 0x02, 0x2c, 0xc5, 0x56, 0xd8, 0xc4, 0x64, 0xd9, 0x21, 0xc1, 0x99, 0x92, 0x93,
	0x4d, 0xaa, 0xa0, 0x98, 0x92, 0x35, 0xed, 0x58, 0x78, 0x46, 0x9a, 0x48, 0x9a, 0x38, 0xe1, 0x89,
	0xa2, 0x8a, 0x47, 0x5e, 0x80, 0x07, 0x0a, 0x6a, 0x5f, 0xf8, 0x05, 0x54, 0xf1, 0xc2, 0x23, 0xfc,
	0x06, 0x1e, 0xf9, 0x17, 0xfc, 0x02, 0xfa, 0x72, 0x7a, 0xd4, 0x2d, 0x69, 0xc6, 0xf6, 0x16, 0x6f,
	0xa3, 0xaf, 0xcf, 0x39, 0x7d, 0xfa, 0xf4, 0xb9, 0xf6, 0xc0, 0x25, 0x6f, 0x1c, 0xec, 0x8c, 0xe3,
	0x28, 0x8d, 0xc8, 0xa5, 0x53, 0x6f, 0x38, 0xa4, 0x69, 0x3c, 0xf6, 0x9d, 0x16, 0x34, 0xbe, 0xa0,
	0x71, 0x12, 0x44, 0xa1, 0x4b, 0x5f, 0x4d, 0x68, 0x92, 0x3a, 0xff, 0xaa, 0x40, 0x73, 0x0a, 0x25,
	0xe3, 0x28, 0x4c, 0x28, 0x79, 0x0f, 0x1a, 0xaf, 0x25, 0xd4, 0x4f, 0xd2, 0x38, 0x08, 0x5f, 0x76,
	0x2a, 0x37, 0x2a, 0xb7, 0x2f, 0xb9, 0x75, 0x44, 0x0f, 0x04, 0x48, 0xd6, 0x61, 0x69, 0xe4, 0xfd,
	0x32, 0x8a, 0x3b, 0x0b, 0x6c, 0xb5, 0xee, 0xca, 0x0f, 0x81, 0x06, 0x21, 0x43, 0x2d, 0x44, 0xf9,
	0x07, 0x47, 0xc7, 0x5e, 0xea, 0x1f, 0x77, 0x16, 0x25, 0x2a, 0x3e, 0xc8, 0x35, 0x80, 0x71, 0x4c,
	0x63, 0x3a, 0xa4, 0x5e, 0x42, 0x3b, 0x4b, 0x62, 0x13, 0x0d, 0xe1, 0x8a, 0x1c, 0x4e, 0x82, 0xe1,
	0xa0, 0x3f, 0xa2, 0xa9, 0x37, 0xf0, 0x52, 0xaf, 0xb3, 0x2c, 0x15, 0x11, 0xe8, 0x4f, 0x11, 0x74,
	0xfe, 0x69, 0x01, 0x79, 0x1a, 0x7b, 0x61, 0xe2, 0xf9, 0x29, 0x53, 0xef, 0x47, 0x0c, 0x0f, 0x86,
	0x09, 0x21, 0xb0, 0x78, 0xec, 0x25, 0xc7, 0x42, 0xf9, 0x9a, 0x2b, 0x7e, 0x93, 0x1b, 0x50, 0x4d,
	0x33, 0x4a, 0xa1, 0x79, 0xcd, 0xd5, 0x21, 0xf2, 0x03, 0x58, 0x1e, 0xd0, 0xc3, 0x20, 0x4d, 0xd8,
	0x01, 0xac, 0xdb, 0xd5, 0xdd, 0x9b, 0x3b, 0x53, 0xf3, 0xed, 0x14, 0x37, 0xd9, 0xe9, 0x86, 0xe3,
	0x49, 0xea, 0x22, 0x0b, 0xb9, 0x07, 0x2b, 0x7e, 0x4c, 0x07, 0x9c, 0x7b, 0x51, 0x70, 0xbf, 0x3b,
	0x9f, 0xfb, 0xc9, 0x24, 0xe5, 0xec, 0x8a, 0x89, 0xb4, 0xc0, 0x3a, 0xa2, 0xd2, 0x12, 0x96, 0xcb,
	0x7f, 0x92, 0xab, 0x70, 0x29, 0x0d, 0x46, 0xec, 0xa6, 0xbc, 0xd1, 0x58, 0x9c, 0xde, 0x72, 0x33,
	0xc0, 0x7e, 0x05, 0x4b, 0x42, 0x01, 0x6e, 0xdf, 0x20, 0x1c, 0xd0, 0x37, 0xe2, 0xb0, 0xcc, 0xbe,
	0xe2, 0x83, 0x7c, 0x00, 0x2d, 0x66, 0xcd, 0xd7, 0x41, 0x34, 0x49, 0xfa, 0x9e, 0xef, 0x47, 0x93,
	0x30, 0xc5, 0xcb, 0x6a, 0x2a, 0xfc, 0xbe, 0x84, 0xc9, 0xfb, 0xd0, 0xcc, 0x48, 0x47, 0x82, 0xd2,
	0x12, 0xbb, 0x35, 0xa6, 0x94, 0x02, 0xb5, 0x9f, 0xc2, 0xb2, 0xd4, 0x7a, 0xc6, 0x9e, 0x1d, 0x58,
	0x31, 0xb7, 0x52, 0x9f, 0xc4, 0x86, 0xd5, 0x20, 0x4c, 0x69, 0x1c, 0x7a, 0x43, 0x21, 0x7b, 0xd5,
	0x9d, 0x7e, 0x3b, 0x7f, 0xa9, 0x40, 0xed, 0xc1, 0x30, 0xf2, 0x4f, 0xe6, 0x5d, 0xde, 0x26, 0x2c,
	0x1f, 0xd3, 0xe0, 0xe5, 0xb1, 0x94, 0xbc, 0xe4, 0xe2, 0x97, 0x69, 0x23, 0x2b, 0x67, 0x23, 0x72,
	0x1f, 0x6a, 0xda, 0xfd, 0xaa, 0x8b, 0xd9, 0x9e, 0x7b, 0x31, 0xae, 0xc1, 0xe2, 0x3c, 0x81, 0x06,
	0xda, 0xe9, 0x81, 0x37, 0xf4, 0x42, 0x9f, 0xea, 0xa7, 0xac, 0x98, 0xa7, 0xbc, 0x09, 0xf5, 0x34,
	0x4a, 0xbd, 0x61, 0xff, 0x50, 0x92, 0x0a, 0x5d, 0x2d, 0x26, 0x90, 0x83, 0xc8, 0xee, 0xd4, 0xa1,
	0xda, 0x63, 0x21, 0xa4, 0x82, 0xb0, 0x01, 0x35, 0xf9, 0x29, 0x03, 0x90, 0x87, 0xe9, 0x3e, 0x4d,
	0x4f, 0xa3, 0xf8, 0x44, 0x51, 0x7c, 0x0c, 0xcd, 0x29, 0x92, 0x45, 0x29, 0xd7, 0xef, 0x35, 0xed,
	0x87, 0x72, 0x05, 0x35, 0xa9, 0x4b, 0x14, 0xc9, 0x9d, 0xef, 0xc3, 0x3a, 0xea, 0xbe, 0x3f, 0x19,
	0x1d, 0xd2, 0x18, 0x25, 0x92, 0x77, 0xa0, 0x86, 0x2a, 0xf7, 0x43, 0x6f, 0x44, 0x31, 0xc4, 0xab,
	0x88, 0xed, 0x33, 0xc8, 0xb9, 0x07, 0x1b, 0x39, 0x56, 0x7d, 0x6b, 0xe4, 0x15, 0x2b, 0xd9, 0xd6,
	0x1a, 0xb9, 0xd3, 0x86, 0x26, 0xf2, 0x27, 0xea, 0x1c, 0xff, 0xb0, 0xa0, 0x95, 0x61, 0x28, 0xee,
	0x87, 0xb0, 0x8a, 0x8c, 0x09, 0x13, 0x94, 0x0f, 0xba, 0x3c, 0xb9, 0x02, 0xdc, 0x29, 0x13, 0xf9,
	0x3a, 0x10, 0x7f, 0x12, 0xc7, 0x94, 0xe9, 0x73, 0xc8, 0x9d, 0xa8, 0x2f, 0x5c, 0x47, 0x06, 0x77,
	0x0b, 0x57, 0x84, 0x77, 0x7d, 0xce, 0xdd, 0xe8, 0x0e, 0xac, 0xe7, 0xa8, 0xa5, 0x53, 0x59, 0xc2,
	0xa9, 0x88, 0x41, 0x2f, 0x56, 0xec, 0xdf, 0x2c, 0xc0, 0x8a, 0x0a, 0x94, 0xf3, 0x9d, 0xbd, 0x60,
	0xde, 0x85, 0x82, 0x79, 0x8b, 0x9e, 0x62, 0x15, 0x3d, 0x85, 0x1f, 0x8d, 0xbe, 0x91, 0x41, 0xd2,
	0x3f, 0xa1, 0x6f, 0xfb, 0xd2, 0xe7, 0x64, 0x16, 0x6d, 0xa9, 0x95, 0x47, 0xf4, 0xed, 0x9e, 0x50,
	0x8e, 0x51, 0xab, 0x90, 0xd2, 0xa8, 0x97, 0x24, 0xb5, 0x5a, 0x31, 0xa8, 0x47, 0xe3, 0x28, 0x4e,
	0xe9, 0x40, 0xa3, 0x5e, 0x46, 0x6a, 0x5c, 0x51, 0xd4, 0xce, 0x0b, 0x58, 0x77, 0x29, 0x3f, 0x8b,
	0xb2, 0x3f, 0x3a, 0xd2, 0x39, 0x0d, 0x72, 0x19, 0x56, 0x43, 0x7a, 0xaa, 0x1b, 0x63, 0x85, 0x7d,
	0x0b, 0x3f, 0xdb, 0x82, 0x8d, 0x9c, 0x64, 0x8c, 0x83, 0xe7, 0x40, 0xf6, 0xd9, 0x19, 0x73, 0x1b,
	0xf2, 0xaa, 0xe1, 0x25, 0xc9, 0xf8, 0x38, 0xe6, 0x55, 0x43, 0x26, 0x08, 0x0d, 0x39, 0x87, 0xe9,
	0x9d, 0x4f, 0x60, 0xcd, 0x10, 0x7c, 0x31, 0xbf, 0xfe, 0x73, 0x05, 0xf5, 0x1a, 0x0c, 0x62, 0x9a,
	0x28, 0xdf, 0x9e, 0x93, 0x13, 0xbe, 0x0b, 0x8b, 0x27, 0x2c, 0x3b, 0x0a, 0x4d, 0x1a, 0xbb, 0x8e,
	0xe6, 0xdc, 0x45, 0x31, 0x3b, 0x8f, 0x18, 0xa5, 0x2b, 0xe8, 0x9d, 0x5d, 0x58, 0xe4, 0x5f, 0x2c,
	0xd3, 0xb6, 0x1e, 0x74, 0x7b, 0x77, 0xee, 0xdc, 0xbd, 0xdb, 0x7f, 0xf8, 0xe2, 0xe9, 0x43, 0x77,
	0xff, 0xfe, 0xe3, 0xd6, 0xd7, 0x74, 0xb4, 0xbb, 0x8f, 0x68, 0xc5, 0xf9, 0x26, 0x1e, 0x4d, 0x09,
	0xc5, 0xa3, 0x71, 0xe5, 0x24, 0x84, 0x91, 0xae, 0x3e, 0x9d, 0x3f, 0x54, 0x60, 0xab, 0x2b, 0x2e,
	0xbb, 0x17, 0x07, 0xaf, 0xbd, 0x94, 0xb2, 0x1b, 0x3f, 0xaf, 0xa9, 0x67, 0x27, 0xfb, 0x5b, 0xbc,
	0x9e, 0x08, 0x71, 0xc2, 0xb5, 0x4e, 0x83, 0x23, 0xe1, 0xde, 0xac, 0x76, 0x8f, 0xa7, 0xbb, 0x3c,
	0x0f, 0x8e, 0x78, 0x4e, 0x67, 0x5a, 0xf8, 0x5e, 0x28, 0x7c, 0x7a, 0xd5, 0xc5, 0x2f, 0xc7, 0x86,
	0x4e, 0x51, 0x29, 0x74, 0x8b, 0x10, 0x1a, 0x18, 0x1e, 0x17, 0xf4, 0xc1, 0xef, 0xc0, 0x66, 0xcc,
	0x38, 0x02, 0x56, 0x6d, 0x99, 0xb3, 0x87, 0x47, 0x41, 0x3c, 0xf2, 0x64, 0x51, 0x90, 0x05, 0x65,
	0x43, 0xad, 0xee, 0xe9, 0x8b, 0x6c, 0xbf, 0xe6, 0x74, 0x3f, 0x34, 0x27, 0xab, 0x7d, 0x22, 0x4c,
	0xc5, 0x3e, 0x96, 0x2b, 0x3f, 0x78, 0x21, 0x4a, 0xc6, 0x34, 0x1c, 0x78, 0x87, 0x43, 0x95, 0xf7,
	0x33, 0x80, 0x97, 0xd8, 0x60, 0xc4, 0x64, 0x4e, 0x62, 0xda, 0x8f, 0xe9, 0xa9, 0x17, 0x0f, 0x54,
	0x89, 0x55, 0xb0, 0x2b, 0x50, 0xe7, 0x4f, 0x0b, 0xb0, 0xf9, 0x63, 0x9a, 0x6a, 0x65, 0x69, 0xea,
	0x63, 0x3b, 0xb0, 0xc6, 0xaa, 0x5a, 0x9c, 0xb2, 0x6a, 0xa1, 0xa7, 0x3a, 0x79, 0x33, 0x6d, 0xb5,
	0x94, 0xe5, 0xba, 0x5d, 0xd8, 0xc8, 0xd3, 0x67, 0x15, 0xb4, 0xed, 0xae, 0x99, 0x1c, 0xb2, 0x9c,
	0x7e, 0x08, 0x6d, 0xa6, 0x72, 0x6e, 0x07, 0x4b, 0xec, 0xd0, 0x94, 0x0b, 0x99, 0x7c, 0xa6, 0x8f,
	0x49, 0x2b, 0xa5, 0x2f, 0x0a, 0x73, 0xb6, 0x75, 0x6a, 0x29, 0xfb, 0x1e, 0x5c, 0x61, 0x0d, 0x61,
	0x30, 0x9a, 0x8c, 0x98, 0x09, 0x7c, 0x9e, 0x82, 0x8d, 0xda, 0xbc, 0x24, 0xf8, 0x2e, 0x23, 0x89,
	0x2b, 0x28, 0x74, 0x33, 0x38, 0x7f, 0x67, 0xce, 0x5a, 0x30, 0x0d, 0xde, 0xc9, 0x67, 0x40, 0x18,
	0x23, 0xbb, 0x5a, 0x43, 0xa4, 0x2c, 0x28, 0x5b, 0x5a, 0xcc, 0xe9, 0x7d, 0x86, 0xdb, 0x16, 0x2c,
	0xba, 0x3c, 0xd2, 0x83, 0xf5, 0x49, 0x58, 0x22, 0x69, 0xe1, 0x3c, 0x8d, 0xc3, 0x1a, 0xb2, 0x1a,
	0x5a, 0xb3, 0x26, 0x7b, 0x6b, 0xef, 0xd8, 0x0b, 0x5f, 0xd2, 0xde, 0x34, 0x76, 0xd4, 0x8d, 0x7e,
	0x0c, 0x16, 0x0b, 0x10, 0x71, 0x83, 0x8d, 0xdd, 0x5b, 0x9a, 0xf0, 0x19, 0x0c, 0x3b, 0x3c, 0x12,
	0x38, 0x0b, 0x77, 0xfa, 0x88, 0xf5, 0xc6, 0x5a, 0x80, 0xca, 0x8a, 0x57, 0x67, 0x68, 0xc6, 0xc6,
	0xc9, 0x78, 0xe2, 0xd5, 0xc8, 0xe4, 0x5d, 0xd6, 0x19, 0x9a, 0x91, 0x39, 0xd7, 0xc0, 0x62, 0x92,
	0x49, 0x15, 0x56, 0x7a, 0x6e, 0xf7, 0x8b, 0xfb, 0x4f, 0x1f, 0xb2, 0x0c, 0x03, 0xb0, 0xdc, 0x7b,
	0xf6, 0xe0, 0x71, 0x77, 0x8f, 0xe5, 0x15, 0x16, 0x90, 0x45, 0x8d, 0x30, 0x20, 0x7f, 0xcd, 0x1c,
	0xf6, 0xb3, 0x49, 0xa8, 0x1f, 0xfa, 0xec, 0xa4, 0xc8, 0xcb, 0x9f, 0x17, 0xbf, 0xa4, 0xa9, 0xea,
	0x37, 0x55, 0xa3, 0x24, 0x40, 0xd9, 0x6d, 0xce, 0x89, 0x58, 0x6b, 0x4e, 0xc4, 0x92, 0x4f, 0xc0,
	0x0e, 0x42, 0x7f, 0x38, 0x19, 0xd0, 0xfe, 0x34, 0xe4, 0xfc, 0x28, 0x08, 0x0f, 0x99, 0xd6, 0x09,
	0x66, 0x9a, 0x0e, 0x52, 0x74, 0x91, 0x60, 0x4f, 0xad, 0xf3, 0xa0, 0x51, 0xdc, 0xbe, 0x38, 0x72,
	0x3f, 0xf1, 0xe3, 0x60, 0x2c, 0x0b, 0xe9, 0xaa, 0xbb, 0x86, 0x8b, 0xd2, 0x1c, 0x07, 0x62, 0xc9,
	0xf9, 0xab, 0x05, 0x5b, 0x05, 0x13, 0xa0, 0x63, 0xfe, 0x1c, 0x5a, 0x09, 0x9b, 0x68, 0x7c, 0x5e,
	0x67, 0x23, 0xd1, 0x3b, 0x2b, 0xb7, 0xfc, 0x96, 0x76, 0xdf, 0x33, 0xb8, 0x77, 0x7a, 0xd8, 0x7f,
	0xe3, 0xac, 0xd0, 0x54, 0xa2, 0xe4, 0x77, 0xc2, 0xcb, 0x9d, 0x6c, 0x23, 0x0c, 0x33, 0x56, 0x05,
	0x86, 0x56, 0xbc, 0x0d, 0x2d, 0x3c, 0xc8, 0xf8, 0x44, 0x9d, 0x45, 0x3a, 0x41, 0x43, 0xe2, 0xbd,
	0x13, 0x79, 0x0c, 0xfb, 0x3f, 0x15, 0x68, 0x98, 0x1b, 0xf2, 0x21, 0x42, 0x0b, 0x03, 0x3d, 0xdf,
	0x34, 0x35, 0x5c, 0x64, 0x03, 0xa6, 0x8a, 0x3c, 0x5f, 0x5f, 0x0e, 0x06, 0xb2, 0x26, 0x54, 0x25,
	0xd6, 0x15, 0xe3, 0x01, 0xcb, 0xf7, 0xc6, 0x78, 0x81, 0x5f, 0xe4, 0x0a, 0x5c, 0xca, 0x74, 0x5b,
	0x14, 0xe2, 0x57, 0xc7, 0xa8, 0x15, 0x97, 0xcb, 0xb3, 0x05, 0xef, 0x75, 0x79, 0x5f, 0x8f, 0xf3,
	0x51, 0x15, 0xb1, 0xa7, 0x81, 0x6c, 0xa6, 0x8e, 0xe2, 0x68, 0x34, 0xbd, 0x65, 0xd1, 0xc6, 0xac,
	0xba, 0x35, 0x0e, 0xaa, 0x9b, 0x75, 0xfe, 0x58, 0x81, 0xcd, 0x83, 0xe0, 0x65, 0x58, 0xe2, 0xa7,
	0x67, 0x55, 0x3a, 0xe6, 0x88, 0x09, 0x8d, 0x03, 0x6f, 0x18, 0xfc, 0xca, 0xcc, 0x0b, 0x18, 0x74,
	0x1b, 0xd9, 0xaa, 0x26, 0x9d, 0xab, 0x15, 0x84, 0x53, 0x83, 0x50, 0x39, 0x54, 0xd6, 0xdd, 0x9a,
	0x00, 0xbb, 0x12, 0x73, 0x5e, 0xc1, 0x56, 0x41, 0x2b, 0x74, 0x9d, 0xdc, 0xbc, 0x5a, 0x29, 0xce,
	0xab, 0x77, 0x61, 0x73, 0x12, 0x26, 0x8c, 0x9d, 0xa9, 0x65, 0x6e, 0xb5, 0x20, 0xb6, 0x5a, 0x57,
	0xab, 0x5d, 0x7d, 0xcb, 0x9f, 0xc0, 0xe5, 0xde, 0xe4, 0x70, 0x18, 0x24, 0xc7, 0x25, 0xb6, 0xf8,
	0x06, 0x10, 0x14, 0x58, 0xdc, 0xbb, 0x2d, 0x57, 0x34, 0x2e, 0xe7, 0x2a, 0xd8, 0x65, 0xb2, 0x30,
	0x37, 0xbc, 0x03, 0xd7, 0x35, 0x78, 0x3f, 0x4a, 0x83, 0xa3, 0xc0, 0xf7, 0xf4, 0xa2, 0xe6, 0x7c,
	0xb9, 0x00, 0x37, 0x66, 0xd3, 0xa0, 0x25, 0x3e, 0x85, 0xa6, 0x97, 0xa6, 0x9e, 0x7f, 0xcc, 0xd4,
	0x12, 0xb5, 0xe6, 0xcc, 0xd4, 0xde, 0x50, 0xf4, 0x02, 0x4d, 0x78, 0xfd, 0x1d, 0x50, 0x53, 0x02,
	0x37, 0x11, 0x0b, 0x02, 0x05, 0x23, 0xe1, 0xac, 0x02, 0x60, 0x7d, 0xd5, 0x02, 0xc0, 0xf3, 0x51,
	0x89, 0x44, 0x11, 0x4b, 0x54, 0x4e, 0xa4, 0x35, 0xb7, 0x53, 0x64, 0xfc, 0x5c, 0xac, 0x3b, 0xbf,
	0xab, 0xc0, 0xf6, 0x01, 0x6b, 0x23, 0xd2, 0x90, 0xf5, 0x6b, 0x65, 0x16, 0x9c, 0x93, 0x65, 0x59,
	0x31, 0x0f, 0xa3, 0x7e, 0xc8, 0x99, 0xde, 0xf6, 0x99, 0x2b, 0x70, 0x31, 0xc2, 0x65, 0x57, 0xdd,
	0x66, 0x18, 0x09, 0x61, 0x6f, 0x9f, 0x49, 0x98, 0xf7, 0x6c, 0x19, 0xad, 0xa4, 0x94, 0x73, 0x7a,
	0x5d, 0x51, 0x0a, 0x2d, 0x9c, 0xdf, 0x2f, 0xc0, 0xb5, 0x59, 0xfa, 0xe0, 0x6d, 0xfd, 0x7f, 0x93,
	0xc6, 0x23, 0x58, 0x11, 0x6d, 0x14, 0x95, 0xaf, 0x4a, 0x66, 0xde, 0x9c, 0xaf, 0x89, 0x58, 0x66,
	0x8c, 0xae, 0x92, 0x60, 0x3f, 0x83, 0x15, 0xc4, 0x2e, 0xa2, 0xe5, 0x75, 0xa8, 0x6a, 0xd1, 0x85,
	0x4a, 0x42, 0x16, 0xc6, 0xce, 0x36, 0x5c, 0x51, 0xc3, 0x72, 0x99, 0x8f, 0xff, 0xb7, 0x02, 0x57,
	0xcb, 0xd7, 0x2f, 0x34, 0x7b, 0x9c, 0x67, 0xae, 0x2c, 0x1f, 0x19, 0xad, 0x0b, 0x8d, 0x8c, 0x8b,
	0x17, 0x1a, 0x19, 0x97, 0x66, 0x8c, 0x8c, 0xbf, 0xad, 0xc0, 0xda, 0x5e, 0x4c, 0x59, 0xfb, 0xfe,
	0x5c, 0x5c, 0x97, 0x72, 0xd7, 0x8f, 0xa0, 0x3d, 0xe6, 0x19, 0xc3, 0xef, 0x17, 0x72, 0x6e, 0x4b,
	0x2e, 0x68, 0xfd, 0x0b, 0xcb, 0x46, 0x6a, 0x92, 0x28, 0xb4, 0x3a, 0x6d, 0x5c, 0xd1, 0xc8, 0x09,
	0x2c, 0x26, 0x94, 0x0e, 0xb0, 0xbe, 0x89, 0xdf, 0xce, 0x26, 0xac, 0x9b, 0x6a, 0x60, 0x6e, 0xfa,
	0x14, 0xda, 0x4f, 0x98, 0x2b, 0x7c, 0x75, 0xe5, 0x9c, 0x75, 0x20, 0xba, 0x04, 0x94, 0xcb, 0xd0,
	0xbd, 0x61, 0x94, 0x98, 0xa7, 0x76, 0x36, 0x98, 0x31, 0x74, 0x14, 0x89, 0x19, 0x2c, 0x91, 0x87,
	0x6f, 0x82, 0x24, 0x7b, 0x29, 0xd9, 0x81, 0x75, 0x13, 0x46, 0x3f, 0x61, 0x05, 0x94, 0x0a, 0x44,
	0xe8, 0xc4, 0x06, 0x26, 0xf9, 0xe5, 0x7c, 0x59, 0x81, 0xce, 0x01, 0xef, 0xe6, 0xf7, 0x38, 0x59,
	0x98, 0x4c, 0x12, 0x77, 0xec, 0xab, 0x33, 0xb1, 0xd4, 0x87, 0x8f, 0x44, 0x7d, 0x73, 0x0a, 0x6c,
	0x20, 0x8c, 0xe3, 0x22, 0x7f, 0xa3, 0x9b, 0x24, 0xfc, 0xca, 0xa7, 0xae, 0x35, 0xfd, 0xe6, 0x6b,
	0xdc, 0x22, 0x8c, 0x5c, 0x59, 0x77, 0xfa, 0xcd, 0xeb, 0x94, 0x4f, 0x63, 0xf4, 0x6b, 0x8a, 0x05,
	0x5c, 0x87, 0x9c, 0x2b, 0x70, 0xb9, 0x44, 0x3d, 0x79, 0xa8, 0x5d, 0x77, 0xfa, 0x2e, 0x7d, 0x40,
	0xe3, 0xd7, 0x81, 0xcf, 0xd3, 0xfd, 0x0a, 0x22, 0xe4, 0xb2, 0x16, 0xec, 0xe6, 0xeb, 0xb5, 0x6d,
	0x97, 0x2d, 0xa1, 0xcc, 0x7f, 0x57, 0xa1, 0x2e, 0x2d, 0xa8, 0x64, 0x7e, 0x0f, 0x16, 0xf9, 0x33,
	0x1b, 0xd9, 0xd4, 0xb8, 0xb4, 0x67, 0x38, 0x7b, 0xab, 0x80, 0x4f, 0x6b, 0xcf, 0x0a, 0x3e, 0xa7,
	0x19, 0xca, 0x98, 0x6f, 0x74, 0x86, 0x32, 0xf9, 0xc7, 0x3a, 0x17, 0xea, 0xc6, 0x53, 0x1a, 0xb9,
	0x5e, 0x7c, 0xe1, 0x32, 0xde, 0xe7, 0xec, 0x1b, 0xb3, 0x09, 0x50, 0xe6, 0x1e, 0xac, 0xaa, 0xb7,
	0x31, 0x62, 0x97, 0x3e, 0x98, 0x49, 0x49, 0x57, 0xe6, 0x3c, 0xa6, 0xf1, 0xa3, 0xa9, 0xa7, 0x26,
	0xfd, 0x68, 0xe6, 0x7c, 0x6d, 0x1c, 0x2d, 0x3f, 0x0a, 0xbf, 0x80, 0x66, 0x6e, 0x22, 0x23, 0xef,
	0x68, 0xe4, 0xe5, 0x83, 0xac, 0xed, 0xcc, 0x23, 0x41, 0xc9, 0x13, 0xe8, 0xcc, 0x6a, 0x0b, 0xc8,
	0x87, 0xe5, 0x55, 0xb8, 0x2c, 0xf7, 0xda, 0x1f, 0x9d, 0x8b, 0x56, 0x6e, 0x7a, 0xa7, 0x42, 0x22,
	0xd6, 0x24, 0x96, 0xd6, 0x14, 0x72, 0xfb, 0x1c, 0x65, 0x47, 0x6e, 0xf9, 0xc1, 0xb9, 0x0b, 0x14,
	0xdb, 0x30, 0xc8, 0x9e, 0x68, 0x8d, 0xed, 0x6e, 0x95, 0xb8, 0x40, 0xd9, 0x66, 0xef, 0x9f, 0x49,
	0x37, 0xdd, 0xea, 0x67, 0xd0, 0xca, 0x4f, 0x71, 0xc4, 0x39, 0x7b, 0xe8, 0xb4, 0x6f, 0xce, 0xa5,
	0xc9, 0x9c, 0xdc, 0x78, 0xc7, 0x33, 0x9c, 0xbc, 0xec, 0xed, 0xd0, 0x70, 0xf2, 0xd2, 0x27, 0x40,
	0xf2, 0x18, 0xaa, 0xda, 0x4b, 0x1d, 0xd9, 0xce, 0xbf, 0x9d, 0x99, 0xf2, 0xae, 0xcd, 0x5a, 0xce,
	0x49, 0xc3, 0x6c, 0xb7, 0x3d, 0xf7, 0x25, 0xae, 0x28, 0x2d, 0xf7, 0xa6, 0xc6, 0x8c, 0x99, 0x7f,
	0xa3, 0x32, 0x8c, 0x39, 0xe3, 0x55, 0xcd, 0x30, 0xe6, 0xac, 0x47, 0x2e, 0x1e, 0x56, 0xb9, 0x89,
	0xd0, 0x08, 0xab, 0xf2, 0x71, 0xdb, 0x08, 0xab, 0x59, 0xe3, 0x28, 0x93, 0x9c, 0x1b, 0x37, 0x0c,
	0xc9, 0xe5, 0x03, 0x92, 0x21, 0x79, 0xd6, 0xb4, 0xe2, 0x01, 0x29, 0x4e, 0x02, 0x44, 0xff, 0x0f,
	0x6c, 0xe6, 0xd0, 0x61, 0xbf, 0x77, 0x06, 0x15, 0x66, 0xf5, 0xbf, 0x59, 0xaa, 0x5c, 0x3e, 0x8e,
	0x3c, 0xd6, 0xc3, 0xa9, 0xdc, 0xfe, 0x04, 0x6a, 0x7a, 0xb9, 0x24, 0xfa, 0xdd, 0x95, 0x94, 0x57,
	0xfb, 0xfa, 0xcc, 0x75, 0x3c, 0x0b, 0x13, 0xa8, 0xf7, 0x0c, 0x86, 0xc0, 0x92, 0x9e, 0xc6, 0x10,
	0x58, 0xd6, 0x6c, 0x90, 0x2e, 0x40, 0xd6, 0x2a, 0x90, 0xab, 0x1a, 0x79, 0xa1, 0x07, 0xb1, 0xb7,
	0x67, 0xac, 0x66, 0x6e, 0xac, 0x75, 0x12, 0x86, 0x1b, 0x17, 0xfb, 0x0e, 0xc3, 0x8d, 0x4b, 0x1a,
	0x10, 0xf2, 0x0b, 0x68, 0x17, 0x2a, 0x33, 0xd1, 0x7d, 0x74, 0x56, 0x5b, 0x61, 0xbf, 0x3b, 0x9f,
	0x48, 0xca, 0x3f, 0x5c, 0x16, 0x7f, 0x43, 0x7f, 0xfb, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x22,
	0xfc, 0x70, 0xb3, 0x93, 0x1e, 0x00, 0x00,
}
