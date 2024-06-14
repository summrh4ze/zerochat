package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type ProfilePanel struct {
	input     component.TextField
	button    widget.Clickable
	OnConfirm func(string)
}

func (profile *ProfilePanel) processEvents(gtx layout.Context) {
	if profile.button.Clicked(gtx) && profile.OnConfirm != nil {
		if profile.input.Text() != "" {
			profile.OnConfirm(profile.input.Text())
		} else {
			profile.input.SetError("Empty nickname")
		}
	}
}

func (profile *ProfilePanel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	profile.processEvents(gtx)
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceAround, Alignment: layout.Middle}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = 600
			return layout.Flex{}.Layout(
				gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					profile.input.CharLimit = 20
					profile.input.MaxLen = 20
					profile.input.Filter = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"
					return profile.input.Layout(gtx, theme, "Enter a nickname")
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Button(theme, &profile.button, "Confirm").Layout(gtx)
					})
				}),
			)
		}),
	)
}
