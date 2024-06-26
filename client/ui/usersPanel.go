package ui

import (
	"example/zerochat/chatProto/domain"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type UsersPanel struct {
	userList  UserList
	selfCard  UserCard
	ConnError bool
}

func CreateUsersPanel(client *domain.Client, changeUserChannel chan<- string) *UsersPanel {
	return &UsersPanel{
		selfCard: UserCard{message: "Your Profile", user: client.User},
		userList: UserList{
			client:            client,
			list:              layout.List{Axis: layout.Vertical},
			changeUserChannel: changeUserChannel,
		},
	}
}

func (up *UsersPanel) processClickEvents(gtx layout.Context) {
	if up.selfCard.btn.Clicked(gtx) {
		up.userList.changeUserChannel <- up.selfCard.user.Id
	}
}

func (up *UsersPanel) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	up.processClickEvents(gtx)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10), Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				var title material.LabelStyle
				if up.ConnError {
					title = material.H6(theme, "You - No Connection")
					title.Color = red
				} else {
					title = material.H6(theme, "You")
				}

				return title.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(30)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return up.selfCard.Layout(gtx, theme)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10), Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.H6(theme, "ONLINE USERS")
				return title.Layout(gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return up.userList.Layout(gtx, theme)
		}),
	)
}
