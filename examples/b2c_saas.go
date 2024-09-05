package examples

import "time"

type SubscriptionPlan struct {
}

type Tenant struct {
	ID       uint   `dba:"title=主键ID"`
	Name     string `dba:"title=租户名称"`
	IsActive *bool  `dba:"title=是否激活"`
}

type Customer struct {
	ID          uint         `dba:"title=主键ID"`
	Username    string       `dba:"title=用户名"`
	Password    string       `dba:"title=密码"`
	Email       string       `dba:"title=邮箱"`
	Phone       string       `dba:"title=电话号码"`
	CountryCode string       `dba:"title=地区代码;desc=例如+1（美国）或 +86（中国）"`
	PhoneNumber string       `dba:"title=电话号码"`
	WalletID    *uint        `dba:"title=客户钱包ID"`
	Wallet      *Wallet      `dba:"title=客户钱包;rel=HAS_ONE,ID->CustomerID"`
	Addresses   []*Address   `dba:"title=收货地址;rel=HAS_MANY,ID->CustomerID"`
	TenantID    uint         `dba:"title=所属租户ID"`
	Tenant      *Tenant      `dba:"title=所属租户;rel=REF_ONE,TenantID->ID"`
	Tags        []*Tag       `dba:"title=已有标签;rel=REF_MANY,customer_tag_refers(id->customer_id,id->tag_id)"` // 直接对表
	Groups      []*UserGroup `dba:"title=已加群组;rel=HAS_MANY,UserGroup(ID->CustomerID,ID->GroupID)"`

	//DefaultAddressID int64    `dba:"title=默认收货地址ID"`
	//DefaultAddress   *Address `dba:"title=默认收货地址;rel=HAS_ONE,ID->CustomerID;rel_filter=IsDefault:true"`
}

type Wallet struct {
	ID         uint  `dba:"title=钱包ID"`
	CustomerID uint  `dba:"title=用户ID"`
	Balance    int64 `dba:"title=余额（分）"`
	Points     int64 `dba:"title=积分"`
}

type Address struct {
	ID          uint   `dba:"title=主键ID"`
	CustomerID  uint   `dba:"title=用户ID"`
	Name        string `dba:"title=收货人姓名"`
	Phone       string `dba:"title=收货人电话"`
	Province    string `dba:"title=省份"`
	City        string `dba:"title=城市"`
	District    string `dba:"title=区/县"`
	AddressLine string `dba:"title=详细地址"`
	PostalCode  string `dba:"title=邮政编码"`
	IsDefault   *bool  `dba:"title=是否默认地址"`
}

type Group struct {
	ID   uint   `dba:"title=群组ID"`
	Name string `dba:"title=群组名称"`
}

type Tag struct {
	ID   uint   `dba:"title=标签ID"`
	Name string `dba:"title=标签名称"`
}

type UserGroup struct {
	CustomerID uint      `dba:"title=用户ID"`
	GroupID    uint      `dba:"title=群组ID"`
	JoinedAt   time.Time `dba:"title=加入时间"`
}
