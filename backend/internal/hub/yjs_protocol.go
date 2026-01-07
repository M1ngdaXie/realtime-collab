package hub

import (
	"encoding/binary"
)

// 这个函数用来从前端发来的二进制数据里，提取出 Yjs 的 ClientID 和对应的 Clock
// data: 前端发来的 []byte
// return: map[clientID]clock
func SniffYjsClientIDs(data []byte) map[uint64]uint64 {
    // 简单的防御：如果不是 Type 1 (Awareness) 消息，直接忽略
    if len(data) < 2 || data[0] != 1 {
        return nil
    }

    result := make(map[uint64]uint64)
    offset := 1 // 跳过 [0] MessageType

    // 读取总长度 (跳过)
    _, n := binary.Uvarint(data[offset:])
    if n <= 0 { return nil }
    offset += n

    // 读取 Count (包含几个客户端的信息)
    count, n := binary.Uvarint(data[offset:])
    if n <= 0 { return nil }
    offset += n

    // 循环读取每个客户端的信息
    for i := 0; i < int(count); i++ {
        if offset >= len(data) { break }

        // 1. 读取 ClientID
        id, n := binary.Uvarint(data[offset:])
        if n <= 0 { break }
        offset += n

        // 2. 读取 Clock (现在我们需要保存它！)
        clock, n := binary.Uvarint(data[offset:])
        if n <= 0 { break }
        offset += n

        // 保存 clientID -> clock
        result[id] = clock

        // 3. 跳过 JSON String
        if offset >= len(data) { break }
        strLen, n := binary.Uvarint(data[offset:])
        if n <= 0 { break }
        offset += n

        // 跳过具体字符串内容
        offset += int(strLen)
    }

    return result
}

// 这个函数用来伪造一条"删除消息"
// 输入：clientClocks map[clientID]clock - 要删除的 ClientID 及其最后已知的 clock 值
// 输出：可以广播给前端的 []byte
func MakeYjsDeleteMessage(clientClocks map[uint64]uint64) []byte {
    if len(clientClocks) == 0 {
        return nil
    }

    // 构造 Payload (负载)
    // 格式: [Count] [ID] [Clock] [StringLen] [String] ...
    payload := make([]byte, 0)

    // 写入 Count (变长整数)
    buf := make([]byte, 10)
    n := binary.PutUvarint(buf, uint64(len(clientClocks)))
    payload = append(payload, buf[:n]...)

    for id, clock := range clientClocks {
        // ClientID
        n = binary.PutUvarint(buf, id)
        payload = append(payload, buf[:n]...)
        // Clock: 必须使用 clock + 1，这样 Yjs 才会接受这个更新！
        n = binary.PutUvarint(buf, clock+1)
        payload = append(payload, buf[:n]...)
        // String Length (填 4，因为 "null" 长度是 4)
        n = binary.PutUvarint(buf, 4)
        payload = append(payload, buf[:n]...)
        // String Content (这里是关键：null 代表删除用户)
        payload = append(payload, []byte("null")...)
    }

    // 构造最终消息: [Type=1] [PayloadLength] [Payload]
    finalMsg := make([]byte, 0)
    finalMsg = append(finalMsg, 1) // Type 1

    n = binary.PutUvarint(buf, uint64(len(payload)))
    finalMsg = append(finalMsg, buf[:n]...) // Length
    finalMsg = append(finalMsg, payload...) // Body

    return finalMsg
}