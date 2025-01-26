package examples

import (
	"context"
	"fmt"
	"github.com/iamdanielyin/dba"
	"time"
)

type Org struct {
	Code string `dba:"name=组织代码;pk"`
	Name string `dba:"name=组织名称;required"`
}

type User struct {
	AuditInfoUser
	AuditInfoTime
	AuditInfoOrg
	ID                  uint       `dba:"name=主键ID;pk;incr"`
	Username            string     `dba:"name=用户名;req"`
	FirstName           string     `dba:"name=名字"`
	LastName            string     `dba:"name=姓氏"`
	Password            string     `dba:"name=密码"`
	Email               string     `dba:"name=邮箱地址;req_group:USER_CONTACT"`
	CountryCode         string     `dba:"name=地区代码"`
	LocalPhoneNumber    string     `dba:"name=本地电话号码"`
	Gender              string     `dba:"name=性别;enum=1:男,2:女,-1:未知;default=-1"`
	Height              float64    `dba:"name=身高（CM）"`
	Weight              float64    `dba:"name=体重（KG）"`
	AllowLogin          *bool      `dba:"name=是否允许界面登录"`
	FirstLoginTime      time.Time  `dba:"name=首次登录时间"`
	LastLoginTime       time.Time  `dba:"name=最后登录时间"`
	PasswordLastUpdated *time.Time `dba:"name=密码最后修改时间;desc=用于提醒用户定期修改密码"`

	LockRiskCount  int        `dba:"name=锁定风控计数;desc=每密码错误+1，解锁时重置"`
	IsLocked       *bool      `dba:"name=是否锁定"`
	LastLockTime   *time.Time `dba:"name=最后锁定时间"`
	LastUnlockerID uint       `dba:"name=最后解锁人ID"`
	LastUnlockTime time.Time  `dba:"name=最后解锁时间"`

	ProfileID *uint        `dba:"name=用户详细信息ID"`
	Profile   *UserProfile `dba:"name=用户详细信息;has_one=UserProfile,sk:ID,dk:UserID"`

	SocialMediaAccounts []*UserSocialMediaAccounts `dba:"name=社交平台账号;has_many=UserSocialMediaAccounts,sk:ID,dk:UserID"`

	OrgCode uint `dba:"name=所属组织代码"`
	Org     *Org `dba:"name=所属组织;ref_one=Org,sk:OrgCode,dk:Code"`

	Tags []*Tag `dba:"name=已有标签;ref_many=UserTagRefers,sk:ID,sk_bridge:UserID,dk:ID,dk_bridge:TagID"` // ref_many:中间表,当前主键,中间表当前主键,目标主键,中间表目标主键

	UserTagRefers []struct {
		UserID uint `dba:"name=用户ID;native:user_id"`
		TagID  uint `dba:"name=标签ID;native:tag_id"`
	} `dba:"ref_many_bridge"`
}

func (User) VirtualFields() map[string]dba.Field {
	return map[string]dba.Field{
		"FullNameCn": {
			Title: "中文姓名",
			Type:  dba.String,
			VirtualHandler: func(ctx context.Context, docPtr any) any {
				u := docPtr.(*User)
				return fmt.Sprintf("%s%s", u.LastName, u.FirstName)
			},
		},
		"FullNameEn": {
			Title: "英文全名",
			Type:  dba.String,
			VirtualHandler: func(ctx context.Context, docPtr any) any {
				u := docPtr.(*User)
				return fmt.Sprintf("%s·%s", u.LastName, u.FirstName)
			},
		},
		"PhoneNumber": {
			Title:         "电话号码",
			Description:   "带加号和地区代码的完整电话号码",
			Example:       "+8613800138000",
			Type:          dba.String,
			RequiredGroup: "USER_CONTACT",
			VirtualHandler: func(ctx context.Context, docPtr any) any {
				u := docPtr.(*User)
				return fmt.Sprintf("+%s%s", u.CountryCode, u.LocalPhoneNumber)
			},
		},
	}
}

type UserSocialMediaAccounts struct {
	ID              uint   `dba:"name=主键;pk;incr"`
	UserID          uint   `dba:"name=关联用户ID"`
	PlatformCode    string `dba:"name=社交平台代码;dict:SOCIAL_MEDIA_PLATFORM"` // TODO 关联系统字典定义
	PlatformAccount string `dba:"name=社交平台账号"`
}

type UserProfile struct {
	UserID   uint   `dba:"name=用户ID;pk"`
	Birthday string `dba:"name=生日（YYYY-MM-DD格式）"`
}

type Tag struct {
	ID   uint   `dba:"name=标签ID;pk;incr"`
	Name string `dba:"name=标签名称;required"`
}

type AuditInfoTime struct {
	CreatedAt time.Time  `dba:"name=创建时间"`
	UpdatedAt time.Time  `dba:"name=最后修改时间"`
	DeletedAt *time.Time `dba:"name=逻辑删除时间"`
}

type AuditInfoUser struct {
	CreatedByID      int    `dba:"name=创建人ID"`
	CreatedByAccount string `dba:"name=创建人账号"`
	CreatedByName    string `dba:"name=创建人名称"`

	UpdatedByID      int    `dba:"name=修改人ID"`
	UpdatedByAccount string `dba:"name=修改人账号"`
	UpdatedByName    string `dba:"name=修改人名称"`

	DeletedByID      int    `dba:"name=删除人ID"`
	DeletedByAccount string `dba:"name=删除人账号"`
	DeletedByName    string `dba:"name=删除人名称"`
}

type AuditInfoOrg struct {
	OrgCode int `dba:"name=组织代码"`
}
