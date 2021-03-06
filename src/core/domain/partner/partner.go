/**
 * Copyright 2014 @ S1N1 Team.
 * name :
 * author : jarryliu
 * date : 2013-12-12 16:55
 * description :
 * history :
 */

package partner

import (
	"errors"
	"fmt"
	"go2o/src/core/domain/interface/partner"
	"go2o/src/core/domain/interface/partner/user"
	userImpl "go2o/src/core/domain/partner/user"
	"go2o/src/core/infrastructure"
	"go2o/src/core/infrastructure/domain"
	"go2o/src/core/variable"
	"time"
)

var _ partner.IPartner = new(Partner)

type Partner struct {
	_value       *partner.ValuePartner
	_saleConf    *partner.SaleConf
	_siteConf    *partner.SiteConf
	_apiInfo     *partner.ApiInfo
	_rep         partner.IPartnerRep
	_shops       []partner.IShop
	_host        string
	_userManager user.IUserManager
	_userRep     user.IUserRep
}

func NewPartner(v *partner.ValuePartner, rep partner.IPartnerRep, userRep user.IUserRep) (partner.IPartner, error) {

	var err error

	if v == nil {
		fmt.Println("-----------", v)
		err = errors.New("101:no such partner")
		return nil, err
	}
	if time.Now().Unix() > v.ExpiresTime {
		err = errors.New("103: partner is expires")
	}

	return &Partner{
		_value:   v,
		_rep:     rep,
		_userRep: userRep,
	}, err
}

func (this *Partner) clearShopCache() {
	this._shops = nil
}

func (this *Partner) GetAggregateRootId() int {
	return this._value.Id
}
func (this *Partner) GetValue() partner.ValuePartner {
	return *this._value
}

func (this *Partner) SetValue(v *partner.ValuePartner) error {
	tv := this._value
	if v.Id == tv.Id {
		tv.Name = v.Name
		tv.Address = v.Address
		if v.LastLoginTime > 0 {
			tv.LastLoginTime = v.LastLoginTime
		}
		if v.LoginTime > 0 {
			tv.LoginTime = v.LoginTime
		}

		if len(v.Logo) != 0 {
			tv.Logo = v.Logo
		}
		tv.Phone = v.Phone
		tv.Pwd = v.Pwd
		tv.UpdateTime = time.Now().Unix()

	}
	return nil
}

// 保存
func (this *Partner) Save() (int, error) {
	var id int = this.GetAggregateRootId()
	if id > 0 {
		return this._rep.SavePartner(this._value)
	}

	return this.createPartner()
}

// 创建商户
func (this *Partner) createPartner() (int, error) {
	if id := this.GetAggregateRootId(); id > 0 {
		return id, nil
	}

	v := this._value
	id, err := this._rep.SavePartner(v)
	if err != nil {
		return id, err
	}

	//todo:事务

	// 初始化商户信息
	this._value.Id = id

	// SiteConf
	this._siteConf = &partner.SiteConf{
		IndexTitle: "线上商店-" + v.Name,
		SubTitle:   "线上商店-" + v.Name,
		Logo:       v.Logo,
		State:      1,
		StateHtml:  "",
	}
	err = this._rep.SaveSiteConf(id, this._siteConf)
	this._siteConf.PartnerId = id

	// SaleConf
	this._saleConf = &partner.SaleConf{
		AutoSetupOrder:  1,
		IntegralBackNum: 0,
	}
	err = this._rep.SaveSaleConf(id, this._saleConf)
	this._saleConf.PartnerId = id

	// 创建API
	this._apiInfo = &partner.ApiInfo{
		ApiId:     domain.NewApiId(id),
		ApiSecret: domain.NewSecret(id),
		WhiteList: "*",
		Enabled:   1,
	}
	err = this._rep.SaveApiInfo(id, this._apiInfo)

	return id, err
}

// 获取商户的域名
func (this *Partner) GetMajorHost() string {
	if len(this._host) == 0 {
		host := this._rep.GetPartnerMajorHost(this.GetAggregateRootId())
		if len(host) == 0 {
			host = fmt.Sprintf("%s.%s", this._value.Usr, infrastructure.GetApp().
				Config().GetString(variable.ServerDomain))
		}

		this._host = host
	}
	return this._host
}

// 获取销售配置
func (this *Partner) GetSaleConf() partner.SaleConf {
	if this._saleConf == nil {

		//10%分成
		//0.2,         #上级
		//0.1,         #上上级
		//0.8          #消费者自己

		this._saleConf = this._rep.GetSaleConf(
			this.GetAggregateRootId())
	}
	return *this._saleConf
}

// 保存销售配置
func (this *Partner) SaveSaleConf(v *partner.SaleConf) error {
	this._saleConf = v
	this._saleConf.PartnerId = this._value.Id
	return this._rep.SaveSaleConf(this.GetAggregateRootId(), this._saleConf)
}

// 获取站点配置
func (this *Partner) GetSiteConf() partner.SiteConf {
	if this._siteConf == nil {
		this._siteConf = this._rep.GetSiteConf(this.GetAggregateRootId())
	}
	return *this._siteConf
}

// 保存站点配置
func (this *Partner) SaveSiteConf(v *partner.SiteConf) error {
	this._siteConf = v
	this._siteConf.PartnerId = this._value.Id
	return this._rep.SaveSiteConf(this.GetAggregateRootId(), this._siteConf)
}

// 获取API信息
func (this *Partner) GetApiInfo() partner.ApiInfo {
	if this._apiInfo == nil {
		this._apiInfo = this._rep.GetApiInfo(this.GetAggregateRootId())
	}
	return *this._apiInfo
}

// 保存API信息
func (this *Partner) SaveApiInfo(v *partner.ApiInfo) error {
	this._apiInfo = v
	this._apiInfo.PartnerId = this._value.Id
	return this._rep.SaveApiInfo(this.GetAggregateRootId(), this._apiInfo)
}

// 新建商店
func (this *Partner) CreateShop(v *partner.ValueShop) partner.IShop {
	v.CreateTime = time.Now().Unix()
	v.PartnerId = this.GetAggregateRootId()
	return newShop(this, v, this._rep)
}

// 获取所有商店
func (this *Partner) GetShops() []partner.IShop {
	if this._shops == nil {
		shops := this._rep.GetShopsOfPartner(this.GetAggregateRootId())
		this._shops = make([]partner.IShop, len(shops))
		for i, v := range shops {
			this._shops[i] = this.CreateShop(v)
		}
	}

	return this._shops
}

// 获取商店
func (this *Partner) GetShop(shopId int) partner.IShop {
	//	v := this.rep.GetValueShop(this.GetAggregateRootId(), shopId)
	//	if v == nil {
	//		return nil
	//	}
	//	return this.CreateShop(v)
	shops := this.GetShops()

	for _, v := range shops {
		if v.GetValue().Id == shopId {
			return v
		}
	}
	return nil
}

// 删除门店
func (this *Partner) DeleteShop(shopId int) error {
	//todo : 检测订单数量
	return this._rep.DeleteShop(this.GetAggregateRootId(), shopId)
}

// 返回用户服务
func (this *Partner) UserManager() user.IUserManager {
	if this._userManager == nil {
		this._userManager = userImpl.NewUserManager(
			this.GetAggregateRootId(),
			this._userRep)
	}
	return this._userManager
}
