package main

// typedef char Int8;
// typedef float Float;
// void WaveOut(void *userdata, Int8 *stream, int len);
import "C"

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

// 这是采样频率
// 44KHz已经很好了
const sampleHz = 44100

// 每秒钟的音符数
const notesPerSecond = 6

// 每个音符的采样数
const samplesPerNote = sampleHz / notesPerSecond

// 是否各通道独立弹奏（否则合并）
var sepChannel = true

// 循环播放？
var loopPlay = false

// 小节号
var barNum = 0

// 小节内第几个音符
var noteNum = 0

// 键的相关参数
var keyParams [1 + 88]struct {
	freq   float64
	dPhase float64
}

// 这些是键的名字
var keyNames = [12]string{"C ", "C#", "D ", "D#", "E ", "F ", "F#", "G ", "G#", "A ", "A#", "B "}

// Key 是键的类型定义
type Key int

// 休止符
const __ = Key(0)

// 同一个八度内的音名
const (
	Cn Key = iota + 4
	Cs
	Dn
	Ds
	En
	Fn
	Fs
	Gn
	Gs
	An
	As
	Bn
)

// K 用来根据音名和组号求其键的编号
func K(key Key, group int) Key {
	if key == __ {
		return 0
	}
	return key + Key((group-1)*12)
}

// 待弹奏的一段五线谱
var notesToPlay = [6][2][16]Key{
	// 第3小节
	{
		{K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0)},
		{K(Bn, 2), K(Bn, 2), K(Fs, 3), K(Fs, 3), K(Bn, 3), K(Bn, 3), K(Fs, 3), K(Fs, 3), K(Gn, 2), K(Gn, 2), K(Dn, 3), K(Dn, 3), K(Gn, 3), K(Gn, 3), K(Dn, 3), K(Dn, 3)},
	},
	// 第4小节
	{
		{K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(An, 4), K(Dn, 5), K(En, 5), K(Fs, 5)},
		{K(An, 2), K(An, 2), K(En, 3), K(En, 3), K(An, 3), K(An, 3), K(En, 3), K(En, 3), K(Dn, 3), K(Dn, 3), K(An, 3), K(An, 3), K(Dn, 4), K(Dn, 4), K(An, 3), K(An, 3)},
	},
	// 第5小节
	{
		{K(En, 5), K(En, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(An, 4), K(Dn, 5), K(En, 5), K(Fs, 5)},
		{K(Bn, 2), K(Bn, 2), K(Fs, 3), K(Fs, 3), K(Bn, 3), K(Bn, 3), K(Fs, 3), K(Fs, 3), K(Gn, 2), K(Gn, 2), K(Dn, 3), K(Dn, 3), K(Gn, 3), K(Gn, 3), K(Dn, 3), K(Dn, 3)},
	},

	// 第6小节
	{
		{K(En, 5), K(En, 5), K(Dn, 5), K(En, 5), K(En, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(An, 4), K(Dn, 5), K(En, 5), K(Fs, 5)},
		{K(An, 2), K(An, 2), K(En, 3), K(En, 3), K(An, 3), K(An, 3), K(En, 3), K(En, 3), K(Dn, 3), K(Dn, 3), K(An, 3), K(An, 3), K(Dn, 4), K(Dn, 4), K(An, 3), K(An, 3)},
	},
	// 第7小节
	{
		{K(En, 5), K(En, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(Dn, 5), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(An, 4), K(Dn, 5), K(En, 5), K(Fs, 5)},
		{K(Bn, 2), K(Bn, 2), K(Fs, 3), K(Fs, 3), K(Bn, 3), K(Bn, 3), K(Fs, 3), K(Fs, 3), K(Gn, 2), K(Gn, 2), K(Dn, 3), K(Dn, 3), K(Gn, 3), K(Gn, 3), K(Dn, 3), K(Dn, 3)},
	},
	// 第8小节
	{
		{K(En, 5), K(En, 5), K(Dn, 5), K(En, 5), K(En, 5), K(An, 5), K(An, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(Fs, 5), K(__, 0), K(__, 0), K(__, 0), K(__, 0)},
		{K(An, 2), K(An, 2), K(En, 3), K(En, 3), K(An, 3), K(An, 3), K(En, 3), K(En, 3), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0), K(__, 0)},
	},
}

// WaveFunc 根据一个相位输出当前波的响应
type WaveFunc func(float64) float64

// PhaseFunc 输入两个相位，输出两个指定格式的音频数据
type PhaseFunc func(float64, float64) (float32, float32)

// 全局生成函数（会在回调里面调用）
var phaseFunc PhaseFunc

// SineWave 正弦波
func SineWave(phase float64) float64 {
	return math.Sin(phase)
}

// LinearWave 线性波
func LinearWave(phase float64) float64 {
	return 1 / math.Pi * phase
}

// TriangleWave 三角波
func TriangleWave(phase float64) float64 {
	switch {
	case phase < math.Pi:
		return 1 / math.Pi * phase
	default:
		// f(phase) == f(2Pi-phase), phase > Pi
		return 1 / math.Pi * (2*math.Pi - phase)
	}
}

// NewPulseWaveFunc 矩形波（方波）
func NewPulseWaveFunc(duty float64) WaveFunc {
	t := 2 * math.Pi * duty
	return func(phase float64) float64 {
		switch {
		case phase < t:
			return 1
		default:
			return -1
		}
	}
}

//export WaveOut
// 这个函数是SDL的音频子系统需要我们提供更多的音频数据的时候回调的
// 注意，这是一个cgo回调函数。注意最开始的函数export声明
func WaveOut(userdata unsafe.Pointer, stream *C.Int8, length C.int) {
	n := int(length)
	hdr := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(stream)), Len: n, Cap: n}
	buf := *(*[]C.Float)(unsafe.Pointer(&hdr))

	// 弹奏已经结束
	if barNum >= len(notesToPlay) {
		return
	}

	// 计算出当前小节的将要弹奏的音符
	barNotes := notesToPlay[barNum]
	key0 := barNotes[0][noteNum]
	key1 := barNotes[1][noteNum]

	var phase0 float64
	var phase1 float64

	// 生成采样数据（核心代码）
	for sample := 0; sample < samplesPerNote; sample++ {
		value0, value1 := phaseFunc(phase0, phase1)

		buf[sample*2+0] = C.Float(value0)
		buf[sample*2+1] = C.Float(value1)

		phase0 += keyParams[key0].dPhase
		phase1 += keyParams[key1].dPhase
		if phase0 >= 2*math.Pi {
			phase0 = 0
		}
		if phase1 >= 2*math.Pi {
			phase1 = 0
		}
	}

	// 打印小节线
	if noteNum%8 == 0 {
		fmt.Println("-----------")
	} else if noteNum%4 == 0 {
		fmt.Println("···········")
	}

	// 打印当前弹奏的音符
	name0 := keyNames[(key0+8)%12]
	name1 := keyNames[(key1+8)%12]
	group0 := '0' + (key0+8)/12
	group1 := '0' + (key1+8)/12

	if key0 == 0 {
		name0 = "--"
		group0 = '-'
	}
	if key1 == 0 {
		name1 = "--"
		group1 = '-'
	}

	fmt.Printf("𝄢:%c%c%c 𝄞:%c%c%c\n",
		name1[0], group1, name1[1],
		name0[0], group0, name0[1],
	)

	// 偏移到下一个音符，为下次弹奏作准备
	if noteNum++; noteNum == 16 {
		if barNum++; barNum >= len(notesToPlay) {
			// 是否需要循环？
			if loopPlay {
				barNum = 0
			}
		}
		noteNum = 0
	}
}

// 计算出第n个键的频率
func keyFreq(n int) float64 {
	return math.Pow(2, float64(n-49)/12.0) * 440
}

// 计算各个键的参数
func initKeyParams() {
	keyParams[0].freq = 0
	keyParams[0].dPhase = 0

	for i := 1; i <= 88; i++ {
		f := keyFreq(i)
		keyParams[i].freq = f
		keyParams[i].dPhase = 2 * math.Pi * f / sampleHz
	}
}

// initPhaseFunc 初始化生成函数
func initPhaseFunc() {
	waveFunc0 := SineWave
	waveFunc1 := SineWave

	phaseFunc = func(phase0 float64, phase1 float64) (float32, float32) {
		var value0, value1 float64
		if !sepChannel {
			value0 = (waveFunc0(phase0) + waveFunc0(phase1)) / 2
			value1 = (waveFunc1(phase0) + waveFunc1(phase1)) / 2
		} else {
			value0 = waveFunc0(phase0)
			value1 = waveFunc1(phase1)
		}
		return float32(value0), float32(value1)
	}
}

// 初始化音频设备
func initAudio() {
	var err error

	// 仅初始化音频子系统
	if err = sdl.Init(sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	// 音频参数
	spec := &sdl.AudioSpec{
		Freq:     sampleHz,      // 采样率(每秒采样数)
		Format:   sdl.AUDIO_F32, // 量化数据格式，这个例子使用浮点类型
		Channels: 2,             // 通道数，立体声。分别对应五线谱的高低音谱表
		Samples:  samplesPerNote,
		Callback: sdl.AudioCallback(C.WaveOut),
	}
	if err = sdl.OpenAudio(spec, nil); err != nil {
		panic(err)
	}
}

func playAndWait() {
	// 开始播放
	sdl.PauseAudio(false)

	// 等待播放完毕
	for barNum < len(notesToPlay) {
		sdl.Delay(1000)
	}

	// 关闭
	sdl.CloseAudio()

	// 退出
	sdl.Quit()
}

func main() {
	initKeyParams()
	initPhaseFunc()
	initAudio()
	playAndWait()
}
