package full

import (
	"open_im_sdk/internal/friend"
	"open_im_sdk/internal/group"
	"open_im_sdk/internal/user"
	"open_im_sdk/open_im_sdk_callback"
	"open_im_sdk/pkg/common"
	"open_im_sdk/pkg/constant"
	sdk "open_im_sdk/pkg/sdk_params_callback"
	api "open_im_sdk/pkg/server_api_params"
	"open_im_sdk/pkg/utils"
)

type Full struct {
	user   *user.User
	friend *friend.Friend
	group  *group.Group
	ch     chan common.Cmd2Value
}

func NewFull(user *user.User, friend *friend.Friend, group *group.Group, ch chan common.Cmd2Value) *Full {
	return &Full{user: user, friend: friend, group: group, ch: ch}
}
func (u *Full) getUsersInfo(callback open_im_sdk_callback.Base, userIDList sdk.GetUsersInfoParam, operationID string) sdk.GetUsersInfoCallback {
	friendList := u.friend.GetDesignatedFriendListInfo(callback, userIDList, operationID)
	blackList := u.friend.GetDesignatedBlackListInfo(callback, userIDList, operationID)
	notIn := make([]string, 0)
	for _, v := range userIDList {
		inFriendList := 0
		for _, friend := range friendList {
			if v == friend.FriendUserID {
				inFriendList = 1
				break
			}
		}
		inBlackList := 0
		for _, black := range blackList {
			if v == black.BlockUserID {
				inBlackList = 1
				break
			}
		}
		if inFriendList == 0 && inBlackList == 0 {
			notIn = append(notIn, v)
		}
	}
	//from svr
	publicList := make([]*api.PublicUserInfo, 0)
	if len(notIn) > 0 {
		publicList = u.user.GetUsersInfoFromSvr(callback, notIn, operationID)
		go func() {
			for _, v := range publicList {
				//Update the faceURL and nickname information of the local chat history with non-friends
				_ = u.user.UpdateMsgSenderFaceURLAndSenderNickname(v.UserID, v.FaceURL, v.Nickname)
				conversationID := utils.GetConversationIDBySessionType(v.UserID, constant.SingleChatType)
				//Update session information of local non-friends
				_, err := u.user.GetConversation(conversationID)
				if err == nil {
					_ = common.TriggerCmdUpdateConversation(common.UpdateConNode{ConID: conversationID, Action: constant.UpdateFaceUrlAndNickName, Args: common.SourceIDAndSessionType{SourceID: v.UserID, SessionType: constant.SingleChatType}}, u.ch)
					_ = common.TriggerCmdUpdateConversation(common.UpdateConNode{ConID: conversationID, Action: constant.ConChange, Args: []string{conversationID}}, u.ch)
				}
			}
		}()
	}
	return common.MergeUserResult(publicList, friendList, blackList)
}
