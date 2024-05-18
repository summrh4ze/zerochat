package ui

import (
	"example/zerochat/client/types"
	"fmt"
	"image/color"
	"sort"
	"strings"

	"gioui.org/layout"
	"gioui.org/widget/material"
)

var blue = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}

type UserList struct {
	previousData []types.UserDetails
	userRegistry *types.Registry
	userCards    []*UserCard
	selected     int
}

func (list *UserList) processClickEvents(gtx layout.Context) {
	for i, card := range list.userCards {
		if card.btn.Clicked(gtx) {
			fmt.Printf("You clicked on item %d\n", i)
			list.selected = i
		}
	}
}

func (list *UserList) updateUserCards() {
	userDetails := list.userRegistry.GetUserDetails()
	sort.Slice(userDetails, func(i, j int) bool {
		return strings.ToLower(userDetails[i].Name) < strings.ToLower(userDetails[j].Name)
	})
	same := true
	if len(list.previousData) > 0 && len(list.previousData) == len(userDetails) {
		for i, user := range userDetails {
			if user.Id != list.previousData[i].Id {
				same = false
				break
			}
		}
	} else {
		same = false
	}

	if !same {
		userDetailsLen := len(userDetails)
		currentCardsLen := cap(list.userCards)

		// increase the capacity in case the number of users grows
		if userDetailsLen > currentCardsLen {
			buf := make([]*UserCard, userDetailsLen-currentCardsLen)
			list.userCards = append(list.userCards, buf...)
		}

		// and rewrite each card to the corresponding user to preserve the order
		for i, user := range userDetails {
			if i < len(list.userCards) {
				if list.userCards[i] == nil {
					list.userCards[i] = &UserCard{
						user:        user,
						displayType: DISPLAY_TYPE_LAST_MESSAGE,
					}
				} else {
					list.userCards[i].user = user
				}
			}
		}

		list.userCards = list.userCards[:len(userDetails)]
		list.previousData = userDetails
	}
}

func (list *UserList) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if list.userRegistry != nil {
		list.updateUserCards()
		list.processClickEvents(gtx)
		usersList := layout.List{Axis: layout.Vertical}
		dim := usersList.Layout(gtx, len(list.userCards), func(gtx layout.Context, index int) layout.Dimensions {
			return list.userCards[index].Layout(gtx, theme)
		})
		return dim
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}

}
