package silk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/wdvxdr1123/go-silk/sdk"

	"modernc.org/libc"
	"modernc.org/libc/sys/types"
)

var (
	ErrInvalid    = errors.New("not a silk stream")
	ErrCodecError = errors.New("codec error")
)

func DecodeSilkBuffToPcm(src []byte, sampleRate int) (dst []byte, err error) {
	var tls = libc.NewTLS()
	reader := bytes.NewBuffer(src)
	f, err := reader.ReadByte()
	if err != nil {
		return
	}
	header := make([]byte, 9)
	var n int
	if f == 2 {
		n, err = reader.Read(header)
		if err != nil {
			return
		}
		if n != 9 {
			err = ErrInvalid
			return
		}
		if string(header) != "#!SILK_V3" {
			err = ErrInvalid
			return
		}
	} else if f == '#' {
		n, err = reader.Read(header)
		if err != nil {
			return
		}
		if n != 8 {
			err = ErrInvalid
			return
		}
		if string(header) != "!SILK_V3" {
			err = ErrInvalid
			return
		}
	} else {
		err = ErrInvalid
		return
	}
	var decControl sdk.SDK_DecControlStruct
	decControl.FAPI_sampleRate = int32(sampleRate)
	decControl.FframesPerPacket = 1
	var decSize int32
	sdk.SDK_Get_Decoder_Size(tls, uintptr(unsafe.Pointer(&decSize)))
	dec := libc.Xmalloc(tls, types.Size_t(decSize))
	defer libc.Xfree(tls, dec)
	if sdk.Init_decoder(tls, dec) != 0 {
		err = ErrCodecError
		return
	}
	// 40ms
	frameSize := sampleRate / 1000 * 40
	in := make([]byte, frameSize)
	buf := make([]byte, frameSize)
	out := &bytes.Buffer{}
	for {
		var nByte int16
		err = binary.Read(reader, binary.LittleEndian, &nByte)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if int(nByte) > frameSize {
			err = ErrInvalid
			return
		}
		n, err = reader.Read(in[:nByte])
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if n != int(nByte) {
			err = ErrInvalid
			return
		}
		sdk.SDK_Decode(tls, dec, uintptr(unsafe.Pointer(&decControl)), 0,
			uintptr(unsafe.Pointer(&in[0])), int32(n),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&nByte)))

		_, _ = out.Write(buf[:nByte*2])

	}
	dst = out.Bytes()
	return
}

func EncodePcmBuffToSilk(src []byte, sampleRate, bitRate int, tencent bool) (dst []byte, err error) {
	var tls = libc.NewTLS()
	var reader = bytes.NewBuffer(src)
	var encControl sdk.SDK_EncControlStruct
	var encStatus sdk.SDK_EncControlStruct
	var packetSizeMs = int32(20)
	{ // default setting
		encControl.FAPI_sampleRate = int32(sampleRate)
		encControl.FmaxInternalSampleRate = 24000
		encControl.FpacketSize = (packetSizeMs * int32(sampleRate)) / 1000
		encControl.FpacketLossPercentage = int32(0)
		encControl.FuseInBandFEC = 0
		encControl.FuseDTX = 0
		encControl.Fcomplexity = 2
		encControl.FbitRate = int32(bitRate)
	}
	var encSizeBytes int32
	ret := sdk.SDK_Get_Encoder_Size(tls, uintptr(unsafe.Pointer(&encSizeBytes)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_create_encoder returned %d", ret)
	}
	psEnc := libc.Xmalloc(tls, types.Size_t(encSizeBytes))
	defer libc.Xfree(tls, psEnc)
	ret = sdk.SDK_InitEncoder(tls, psEnc, uintptr(unsafe.Pointer(&encStatus)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_reset_encoder returned %d", ret)
	}
	var frameSize = sampleRate / 1000 * 40
	fmt.Printf("包长:%v", frameSize)
	var (
		nBytes  = int16(250 * 5)
		in      = make([]byte, frameSize)
		payload = make([]byte, nBytes)
		out     = bytes.Buffer{}
	)
	if tencent {
		_, _ = out.Write([]byte("\x02#!SILK_V3"))
	} else {
		_, _ = out.Write([]byte("#!SILK_V3"))
	}
	var counter int
	for {
		counter, err = reader.Read(in)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if counter < frameSize {
			break
		}
		nBytes = int16(1250)
		ret = sdk.SDK_Encode(
			tls,
			psEnc,
			uintptr(unsafe.Pointer(&encControl)),
			uintptr(unsafe.Pointer(&in[0])),
			int32(counter)/2,
			uintptr(unsafe.Pointer(&payload[0])),
			uintptr(unsafe.Pointer(&nBytes)),
		)

		if ret != 0 {
			return nil, fmt.Errorf("SKP_Silk_Encode returned %d", ret)
		}
		_ = binary.Write(&out, binary.LittleEndian, nBytes)
		_, _ = out.Write(payload[:nBytes])
	}
	if !tencent {
		_ = binary.Write(&out, binary.LittleEndian, int16(-1))
	}
	dst = out.Bytes()
	return
}

// Assuming necessary imports and definitions

func EncodePcmBuffToSilkv2(src []byte, sampleRate, bitRate int, tencent, bigEndian bool, complexityMode int) (dst []byte, err error) {
	var tls = libc.NewTLS()
	var reader = bytes.NewBuffer(src)
	var encControl sdk.SDK_EncControlStruct
	var encStatus sdk.SDK_EncControlStruct
	var packetSizeMs = int32(20)
	var frameSizeReadFromFileMs = int32(20)

	// Setting based on the complexity mode (could be passed as a parameter)
	encControl.Fcomplexity = int32(complexityMode)

	// Default settings
	encControl.FAPI_sampleRate = int32(sampleRate)
	encControl.FmaxInternalSampleRate = 24000
	encControl.FpacketSize = (packetSizeMs * int32(sampleRate)) / 1000
	encControl.FpacketLossPercentage = int32(0)
	encControl.FuseInBandFEC = 0
	encControl.FuseDTX = 0
	encControl.FbitRate = int32(bitRate)

	// Create Encoder
	var encSizeBytes int32
	ret := sdk.SDK_Get_Encoder_Size(tls, uintptr(unsafe.Pointer(&encSizeBytes)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_create_encoder returned %d", ret)
	}

	// Memory management
	psEnc := libc.Xmalloc(tls, types.Size_t(encSizeBytes))
	defer libc.Xfree(tls, psEnc)

	// Reset Encoder
	ret = sdk.SDK_InitEncoder(tls, psEnc, uintptr(unsafe.Pointer(&encStatus)))
	if ret != 0 {
		return nil, fmt.Errorf("SKP_Silk_reset_encoder returned %d", ret)
	}

	var (
		nBytes  = int16(250 * 5)
		in      = make([]byte, frameSizeReadFromFileMs*int32(sampleRate)/1000)
		payload = make([]byte, nBytes)
		out     = bytes.Buffer{}
	)

	// Add Silk header to stream
	if tencent {
		_, _ = out.Write([]byte("\x02#!SILK_V3"))
	} else {
		_, _ = out.Write([]byte("#!SILK_V3"))
	}

	// Encoding loop
	var counter int
	for {
		counter, err = reader.Read(in)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}

		if bigEndian {
			SwapEndian(in)
		}

		nBytes = int16(1250)
		ret = sdk.SDK_Encode(
			tls,
			psEnc,
			uintptr(unsafe.Pointer(&encControl)),
			uintptr(unsafe.Pointer(&in[0])),
			int32(counter)/2,
			uintptr(unsafe.Pointer(&payload[0])),
			uintptr(unsafe.Pointer(&nBytes)),
		)

		if ret != 0 {
			return nil, fmt.Errorf("SKP_Silk_Encode returned %d", ret)
		}

		_ = binary.Write(&out, binary.LittleEndian, nBytes)
		_, _ = out.Write(payload[:nBytes])
	}

	if !tencent {
		_ = binary.Write(&out, binary.LittleEndian, int16(-1))
	}

	dst = out.Bytes()
	return
}

func SwapEndian(data []byte) {
	if len(data)%2 != 0 {
		panic("SwapEndian requires an even length byte slice")
	}

	for i := 0; i < len(data); i += 2 {
		data[i], data[i+1] = data[i+1], data[i]
	}
}

