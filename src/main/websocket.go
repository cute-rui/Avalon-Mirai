package main

import (
    "github.com/gorilla/websocket"
    jsoniter "github.com/json-iterator/go"
    "log"
)

/*type WebsocketInitEvent struct {
    SyncId string `json:"syncId"`
    Data   struct {
        Code    int    `json:"code"`
        Session string `json:"session"`
    } `json:"data"`
}*/

type WebsocketEvent struct {
    SyncId string               `json:"syncId"`
    Data   *jsoniter.RawMessage `json:"data"`
}

type WebsocketParam struct {
    SyncId     string              `json:"syncId"`
    Command    string              `json:"command"`
    SubCommand string              `json:"subCommand,omitempty"`
    Content    jsoniter.RawMessage `json:"content,omitempty"`
}

func (s *WebsocketServer) StartWebsocket(url string) error {
    c, _, err := websocket.DefaultDialer.Dial(url, nil)
    if err != nil {
        log.Println(err)
        return err
    }
    
    defer c.Close()
    s.Running = true
    
    go func() {
        for {
            select {
            case action := <-s.InputChan:
                err = c.WriteMessage(1, action)
                if err != nil {
                    log.Println(err)
                }
            }
        }
    }()
    
    _, _, err = c.ReadMessage()
    if err != nil {
        log.Println(err)
    }
    
    for {
        _, msg, err := c.ReadMessage()
        if err != nil {
            log.Println(err)
            continue
        }
        
        id, msg, err := Unwarp(msg)
        if err != nil {
            log.Println(err)
            continue
        }
        
        i, ok := s.OutputChan.Load(id)
        if !ok {
            continue
        }
        
        if ch, ok := i.(chan []byte); ok {
            ch <- msg
        } else {
            s.OutputChan.Delete(id)
        }
    }
    
}

/*func CheckStatus(resp *http.Response) bool {
    respBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Println(err)
        return false
    }

    var e WebsocketInitEvent
    err = jsoniter.Unmarshal(respBytes, &e)
    if err != nil {
        log.Println(err)
        return false
    }

    if e.Data.Code == 0 {
        return true
    }

    return false
}*/

func Unwarp(msg []byte) (string, []byte, error) {
    var e WebsocketEvent
    
    err := jsoniter.Unmarshal(msg, &e)
    if err != nil {
        return "", nil, err
    }
    
    return e.SyncId, *e.Data, nil
}
