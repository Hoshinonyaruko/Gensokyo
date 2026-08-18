package v2

import (
	"context"
	"strconv"

	"github.com/tencent-connect/botgo/dto"
)

// GetGlobalMenu 获取当前生效的全局自定义菜单.
func (o *openAPIv2) GetGlobalMenu(ctx context.Context) (*dto.Menu, error) {
	rsp, err := o.request(ctx).
		SetResult(dto.MenuWrapper{}).
		Get(o.getURL(globalMenuURI))
	if err != nil {
		return nil, err
	}

	result := rsp.Result().(*dto.MenuWrapper)
	return result.Menu, nil
}

// UpdateGlobalMenu 全量更新全局自定义菜单.
// menu == nil 时请求体序列化为 {}，用于清空菜单.
func (o *openAPIv2) UpdateGlobalMenu(ctx context.Context, menu *dto.Menu) error {
	_, err := o.request(ctx).
		SetBody(dto.MenuWrapper{
			Menu: menu,
		}).
		Put(o.getURL(globalMenuURI))

	return err
}

// ListPanels 分页查询指定场景的指令面板.
func (o *openAPIv2) ListPanels(
	ctx context.Context,
	scope string,
	cursor string,
	limit int,
) (*dto.PanelsResponse, error) {
	req := o.request(ctx).
		SetResult(dto.PanelsResponse{}).
		SetQueryParam("scope", scope)

	if cursor != "" {
		req.SetQueryParam("cursor", cursor)
	}

	if limit > 0 {
		req.SetQueryParam("limit", strconv.Itoa(limit))
	}

	rsp, err := req.Get(o.getURL(panelsURI))
	if err != nil {
		return nil, err
	}

	return rsp.Result().(*dto.PanelsResponse), nil
}

// CreatePanel 创建指令面板.
func (o *openAPIv2) CreatePanel(
	ctx context.Context,
	value *dto.PanelToCreate,
) (*dto.PanelCreateResponse, error) {
	rsp, err := o.request(ctx).
		SetResult(dto.PanelCreateResponse{}).
		SetBody(value).
		Post(o.getURL(panelsURI))
	if err != nil {
		return nil, err
	}

	return rsp.Result().(*dto.PanelCreateResponse), nil
}

// GetPanel 获取指定指令面板详情.
func (o *openAPIv2) GetPanel(
	ctx context.Context,
	panelID string,
) (*dto.PanelRecord, error) {
	rsp, err := o.request(ctx).
		SetResult(dto.PanelRecord{}).
		SetPathParam("panel_id", panelID).
		Get(o.getURL(panelURI))
	if err != nil {
		return nil, err
	}

	return rsp.Result().(*dto.PanelRecord), nil
}

// UpdatePanel 全量更新指定指令面板内容.
// target 关联对象不在本接口修改.
func (o *openAPIv2) UpdatePanel(
	ctx context.Context,
	panelID string,
	panel *dto.Panel,
) error {
	_, err := o.request(ctx).
		SetPathParam("panel_id", panelID).
		SetBody(dto.PanelWrapper{
			Panel: panel,
		}).
		Put(o.getURL(panelURI))

	return err
}

// DeletePanel 删除指定指令面板.
func (o *openAPIv2) DeletePanel(
	ctx context.Context,
	panelID string,
) error {
	_, err := o.request(ctx).
		SetPathParam("panel_id", panelID).
		Delete(o.getURL(panelURI))

	return err
}

// UpdatePanelTargets 增加或删除 specific 指令面板关联对象.
func (o *openAPIv2) UpdatePanelTargets(
	ctx context.Context,
	panelID string,
	value *dto.PanelTargetToUpdate,
) error {
	_, err := o.request(ctx).
		SetPathParam("panel_id", panelID).
		SetBody(value).
		Put(o.getURL(panelTargetURI))

	return err
}