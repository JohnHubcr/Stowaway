package common

import (
	"Stowaway/config"
	"Stowaway/crypto"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"

	"github.com/sirupsen/logrus"
)

type Command struct {
	NodeId uint32 //节点序号

	CommandLength uint32 //命令长度

	Command string //命令类型

	InfoLength uint32 //载荷长度

	Info string //具体载荷
}

type Data struct {
	NodeId uint32 //节点序号

	Clientsocks uint32 //socks标号

	Success string //保留此字段，为后续功能留用

	DatatypeLength uint32 //数据类型长度

	Datatype string //数据类型

	ResultLength uint32 //具体载荷长度

	Result string //具体载荷
}

func ExtractCommand(conn net.Conn, key []byte) (*Command, error) {
	var (
		command    = &Command{}
		idlen      = make([]byte, config.ID_LEN)
		commandlen = make([]byte, config.HEADER_LEN)
	)
	if len(key) != 0 {
		key, _ = crypto.KeyPadding(key)
	}

	_, err := io.ReadFull(conn, idlen)
	if err != nil {
		return command, err
	}

	command.NodeId = binary.BigEndian.Uint32(idlen)

	_, err = io.ReadFull(conn, commandlen)
	if err != nil {
		return command, err
	}

	command.CommandLength = binary.BigEndian.Uint32(commandlen)

	commandbuffer := make([]byte, command.CommandLength)
	_, err = io.ReadFull(conn, commandbuffer)
	if err != nil {
		return command, err
	}
	if len(key) != 0 {
		command.Command = string(crypto.AESDecrypt(commandbuffer[:], key))
	} else {
		command.Command = string(commandbuffer[:])
	}

	infolen := make([]byte, config.INFO_LEN)
	_, err = io.ReadFull(conn, infolen)
	if err != nil {
		return command, err
	}
	command.InfoLength = binary.BigEndian.Uint32(infolen)

	infobuffer := make([]byte, command.InfoLength)
	_, err = io.ReadFull(conn, infobuffer)
	if err != nil {
		return command, err
	}
	if len(key) != 0 {
		command.Info = string(crypto.AESDecrypt(infobuffer[:], key))
	} else {
		command.Info = string(infobuffer[:])
	}

	return command, nil

}

func ConstructCommand(command string, info string, id uint32, key []byte) ([]byte, error) {
	var buffer bytes.Buffer

	InfoLength := make([]byte, 5)
	CommandLength := make([]byte, 4)
	Nodeid := make([]byte, 4)

	Command := []byte(command)
	Info := []byte(info)

	if len(key) != 0 {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		Command = crypto.AESEncrypt(Command, key)
		Info = crypto.AESEncrypt(Info, key)
	}

	binary.BigEndian.PutUint32(Nodeid, id)
	binary.BigEndian.PutUint32(CommandLength, uint32(len(Command)))
	binary.BigEndian.PutUint32(InfoLength, uint32(len(Info)))

	buffer.Write(Nodeid)
	buffer.Write(CommandLength)
	buffer.Write(Command)
	buffer.Write(InfoLength)
	buffer.Write(Info)

	final := buffer.Bytes()

	return final, nil

}

func ConstructDataResult(nodeid uint32, clientsocks uint32, success string, datatype string, result string, key []byte) ([]byte, error) {
	var buffer bytes.Buffer
	NodeIdLength := make([]byte, 4)
	ClientsocksLength := make([]byte, 20)
	DatatypeLength := make([]byte, 5)
	ResultLength := make([]byte, 512)

	Success := []byte(success)
	Datatype := []byte(datatype)
	Result := []byte(result)

	if len(key) != 0 {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		Datatype = crypto.AESEncrypt(Datatype, key)
		Result = crypto.AESEncrypt(Result, key)
	}

	binary.BigEndian.PutUint32(NodeIdLength, nodeid)
	binary.BigEndian.PutUint32(ClientsocksLength, uint32(clientsocks))
	binary.BigEndian.PutUint32(DatatypeLength, uint32(len(Datatype)))
	binary.BigEndian.PutUint32(ResultLength, uint32(len(Result)))

	buffer.Write(NodeIdLength)
	buffer.Write(ClientsocksLength)
	buffer.Write(Success)
	buffer.Write(DatatypeLength)
	buffer.Write(Datatype)
	buffer.Write(ResultLength)
	buffer.Write(Result)

	final := buffer.Bytes()

	return final, nil
}

func ExtractDataResult(conn net.Conn, key []byte) (*Data, error) {
	var (
		data        = &Data{}
		nodelen     = make([]byte, config.NODE_LEN)
		clientlen   = make([]byte, config.CLIENT_LEN)
		successlen  = make([]byte, config.SUCCESS_LEN)
		datatypelen = make([]byte, config.DATATYPE_LEN)
		resultlen   = make([]byte, config.RESULT_LEN)
	)

	if len(key) != 0 {
		key, _ = crypto.KeyPadding(key)
	}

	_, err := io.ReadFull(conn, nodelen)
	if err != nil {
		return data, err
	}

	data.NodeId = binary.BigEndian.Uint32(nodelen)

	_, err = io.ReadFull(conn, clientlen)
	if err != nil {
		return data, err
	}

	data.Clientsocks = binary.BigEndian.Uint32(clientlen)

	_, err = io.ReadFull(conn, successlen)
	if err != nil {
		return data, err
	}

	data.Success = string(successlen[:])

	_, err = io.ReadFull(conn, datatypelen)
	if err != nil {
		return data, err
	}
	data.DatatypeLength = binary.BigEndian.Uint32(datatypelen)

	datatypebuffer := make([]byte, data.DatatypeLength)
	_, err = io.ReadFull(conn, datatypebuffer)
	if err != nil {
		return data, err
	}
	if len(key) != 0 {
		data.Datatype = string(crypto.AESDecrypt(datatypebuffer[:], key))
	} else {
		data.Datatype = string(datatypebuffer[:])
	}

	_, err = io.ReadFull(conn, resultlen)
	if err != nil {
		return data, err
	}
	data.ResultLength = binary.BigEndian.Uint32(resultlen)

	resultbuffer := make([]byte, data.ResultLength)
	_, err = io.ReadFull(conn, resultbuffer)
	if err != nil {
		return data, err
	}
	if len(key) != 0 {
		data.Result = string(crypto.AESDecrypt(resultbuffer[:], key))
	} else {
		data.Result = string(resultbuffer[:])
	}

	return data, nil
}