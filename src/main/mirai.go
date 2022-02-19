package main

import (
    "avalon-mirai/src/mirai"
    "context"
    jsoniter "github.com/json-iterator/go"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/encoding/protojson"
    "log"
    "strconv"
    "sync"
    "time"
)

const (
    ABOUT                                     = `about`
    MESSAGE_FROM_ID                           = `messageFromId`
    FRIEND_LIST                               = `friendList`
    GROUP_LIST                                = `groupList`
    MEMBER_LIST                               = `memberList`
    BOT_PROFILE                               = `botProfile`
    FRIEND_PROFILE                            = `friendProfile`
    MEMBER_PROFILE                            = `memberProfile`
    SEND_FRIEND_MESSAGE                       = `sendFriendMessage`
    SEND_GROUP_MESSAGE                        = `sendGroupMessage`
    SEND_TEMP_MESSAGE                         = `sendTempMessage`
    SEND_NUDGE                                = `sendNudge`
    RECALL                                    = `recall`
    FILE_LIST                                 = `file_list`
    FILE_INFO                                 = `file_info`
    FILE_MKDIR                                = `file_mkdir`
    FILE_DELETE                               = `file_delete`
    FILE_MOVE                                 = `file_move`
    FILE_RENAME                               = `file_rename`
    DELETE_FRIEND                             = `deleteFriend`
    MUTE                                      = `mute`
    UNMUTE                                    = `unmute`
    KICK                                      = `kick`
    QUIT                                      = `quit`
    MUTE_ALL                                  = `muteAll`
    UNMUTE_ALL                                = `unmuteAll`
    SET_ESSENCE                               = `setEssence`
    GROUP_CONFIG                              = `groupConfig`
    MEMBERINFO                                = `memberInfo`
    MEMBERADMIN                               = `memberAdmin`
    RESP_NEW_FRIEND_REQUEST_EVENT             = `resp_newFriendRequestEvent`
    RESP_MEMBER_JOIN_REQUEST_EVENT            = `resp_memberJoinRequestEvent`
    RESP_BOT_INVITED_JOIN_GROUP_REQUEST_EVENT = `resp_botInvitedJoinGroupRequestEvent`
)

const (
    GET    = `get`
    UPDATE = `update`
)

type MiraiAgentServiceServer struct {
    WebsocketServers *sync.Map
    mirai.UnimplementedMiraiAgentServer
}

type WebsocketServer struct {
    Running    bool
    InputChan  chan []byte
    OutputChan *sync.Map
}

func (m *MiraiAgentServiceServer) Subscribe(param *mirai.InitParam, server mirai.MiraiAgent_SubscribeServer) error {
    var ch chan []byte
    
    i, ok := m.WebsocketServers.Load(param.GetQq())
    if !ok {
        WSServer := WebsocketServer{
            InputChan:  make(chan []byte, 2),
            OutputChan: &sync.Map{},
        }
        
        ch = make(chan []byte, 2)
        WSServer.OutputChan.Store(Conf.GetString(`mirai.server.websocketServerSideSyncId`), ch)
        
        url := StringBuilder(`ws://`, Conf.GetString(`mirai.server.addr`), `/`, param.GetMessageChannel().String(), `?verifyKey=`, param.GetVerifyKey(), `&qq=`, strconv.Itoa(int(param.GetQq())))
        
        m.WebsocketServers.Store(param.GetQq(), &WSServer)
        go WSServer.StartWebsocket(url)
    } else {
        s := i.(*WebsocketServer)
        i, _ = s.OutputChan.Load(Conf.GetString(`mirai.server.websocketServerSideSyncId`))
        ch = i.(chan []byte)
    }
    
    for {
        select {
        case e := <-ch:
            go func(b []byte) {
                var m mirai.Message
                err := protojson.Unmarshal(b, &m)
                if err != nil {
                    log.Println(err)
                    return
                }
                
                err = server.Send(&m)
                if err != nil {
                    log.Println(err)
                    return
                }
            }(e)
        case <-server.Context().Done():
            return nil
        }
    }
}

func (m *MiraiAgentServiceServer) About(ctx context.Context, param *mirai.SelfParam) (*mirai.AboutResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: ABOUT,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.AboutResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetMessageFromId(ctx context.Context, param *mirai.GetMessageParam) (*mirai.GetMessageResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MESSAGE_FROM_ID,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetMessageResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) ListFriend(ctx context.Context, param *mirai.SelfParam) (*mirai.ListFriendResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FRIEND_LIST,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.ListFriendResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
}

func (m *MiraiAgentServiceServer) ListGroup(ctx context.Context, param *mirai.SelfParam) (*mirai.ListGroupResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: GROUP_LIST,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.ListGroupResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) ListMember(ctx context.Context, param *mirai.ListMemberParam) (*mirai.ListMemberResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MEMBER_LIST,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.ListMemberResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
}

func (m *MiraiAgentServiceServer) GetBotProfile(ctx context.Context, param *mirai.SelfParam) (*mirai.GetProfileResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: BOT_PROFILE,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetProfileResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetFriendProfile(ctx context.Context, param *mirai.GetFriendParam) (*mirai.GetProfileResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FRIEND_PROFILE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetProfileResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetMemberProfile(ctx context.Context, param *mirai.GetMemberParam) (*mirai.GetProfileResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MEMBER_PROFILE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetProfileResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendFriendMessage(ctx context.Context, param *mirai.SendFriendMessageParam) (*mirai.UniversalSendMessageResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: SEND_FRIEND_MESSAGE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalSendMessageResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendGroupMessage(ctx context.Context, param *mirai.SendGroupMessageParam) (*mirai.UniversalSendMessageResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: SEND_GROUP_MESSAGE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalSendMessageResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendTempMessage(ctx context.Context, param *mirai.SendTempMessageParam) (*mirai.UniversalSendMessageResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: SEND_TEMP_MESSAGE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalSendMessageResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendNudge(ctx context.Context, param *mirai.SendNudgeParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: SEND_NUDGE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) Recall(ctx context.Context, param *mirai.RecallParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: RECALL,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) ListFile(ctx context.Context, param *mirai.ListFileParam) (*mirai.ListFileResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_LIST,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.ListFileResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetFileInfo(ctx context.Context, param *mirai.GetFileInfoParam) (*mirai.GetFileInfoResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_INFO,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetFileInfoResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) FileMkdir(ctx context.Context, param *mirai.FileMkdirParam) (*mirai.FileMkdirResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_MKDIR,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.FileMkdirResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) FileDelete(ctx context.Context, param *mirai.FileDeleteParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_DELETE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) FileMove(ctx context.Context, param *mirai.FileMoveParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_MOVE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) FileRename(ctx context.Context, param *mirai.FileRenameParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: FILE_RENAME,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) DeleteFriend(ctx context.Context, param *mirai.DeleteFriendParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: DELETE_FRIEND,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) Mute(ctx context.Context, param *mirai.MuteParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MUTE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) Unmute(ctx context.Context, param *mirai.UnmuteParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: UNMUTE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) Kick(ctx context.Context, param *mirai.KickParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: KICK,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) Quit(ctx context.Context, param *mirai.QuitParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: QUIT,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) MuteAll(ctx context.Context, param *mirai.MuteAllParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MUTE_ALL,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) UnmuteAll(ctx context.Context, param *mirai.UnmuteAllParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: UNMUTE_ALL,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SetEssence(ctx context.Context, param *mirai.SetEssenceParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: SET_ESSENCE,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetGroupConfig(ctx context.Context, param *mirai.GetGroupConfigParam) (*mirai.GetGroupConfigResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:     token,
        Command:    GROUP_CONFIG,
        SubCommand: GET,
        Content:    data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetGroupConfigResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) UpdateGroupConfig(ctx context.Context, param *mirai.UpdateGroupConfigParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:     token,
        Command:    GROUP_CONFIG,
        SubCommand: UPDATE,
        Content:    data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) GetMemberInfo(ctx context.Context, param *mirai.GetMemberInfoParam) (*mirai.GetMemberInfoResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:     token,
        Command:    MEMBERINFO,
        SubCommand: GET,
        Content:    data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.GetMemberInfoResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) UpdateMemberInfo(ctx context.Context, param *mirai.UpdateMemberInfoParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:     token,
        Command:    MEMBERINFO,
        SubCommand: UPDATE,
        Content:    data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SetMemberAdmin(ctx context.Context, param *mirai.SetMemberAdminParam) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(param.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    param.BotQQNumber = nil
    data, err := protojson.Marshal(param)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: MEMBERADMIN,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

type responseFriend struct {
    EventId int64  `json:"eventId"`
    FromId  int64  `json:"fromId"`
    GroupId int64  `json:"groupId"`
    Operate int32  `json:"operate"`
    Message string `json:"message"`
}

func (m *MiraiAgentServiceServer) SendNewFriendRequestEventResponse(ctx context.Context, response *mirai.NewFriendRequestEventResponse) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(response.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    response.BotQQNumber = nil
    
    data, err := jsoniter.Marshal(&responseFriend{
        EventId: response.GetEventId(),
        FromId:  response.GetFromId(),
        GroupId: response.GetGroupId(),
        Operate: response.GetOperate(),
        Message: response.GetMessage(),
    })
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: RESP_NEW_FRIEND_REQUEST_EVENT,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendMemberJoinRequestEventResponse(ctx context.Context, response *mirai.MemberJoinRequestEventResponse) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(response.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    response.BotQQNumber = nil
    data, err := protojson.Marshal(response)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: RESP_MEMBER_JOIN_REQUEST_EVENT,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}

func (m *MiraiAgentServiceServer) SendBotInvitedJoinGroupRequestEventResponse(ctx context.Context, response *mirai.BotInvitedJoinGroupRequestEventResponse) (*mirai.UniversalResponseResult, error) {
    i, ok := m.WebsocketServers.Load(response.GetBotQQNumber())
    if !ok {
        return nil, status.Error(codes.NotFound, `Could not found bot`)
    }
    
    s := i.(*WebsocketServer)
    
    response.BotQQNumber = nil
    data, err := protojson.Marshal(response)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    token := RandString(10)
    p := WebsocketParam{
        SyncId:  token,
        Command: RESP_BOT_INVITED_JOIN_GROUP_REQUEST_EVENT,
        Content: data,
    }
    
    b, err := jsoniter.Marshal(p)
    if err != nil {
        log.Println(err)
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    ch := make(chan []byte)
    s.OutputChan.Store(token, ch)
    
    s.InputChan <- b
    
    select {
    case resp := <-ch:
        s.OutputChan.Delete(token)
        var m mirai.UniversalResponseResult
        err := protojson.Unmarshal(resp, &m)
        if err != nil {
            log.Println(err)
            return nil, status.Error(codes.Internal, err.Error())
        }
        
        return &m, nil
    case <-time.After(time.Second * 300):
        s.OutputChan.Delete(token)
        return nil, status.Error(codes.Internal, StringBuilder(token, `pipe time out`))
    case <-ctx.Done():
        s.OutputChan.Delete(token)
        return nil, nil
    }
    
}
