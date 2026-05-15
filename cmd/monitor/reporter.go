package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libmetric"
	"github.com/pointnoreturn/monitor/libweather"
	"github.com/pointnoreturn/monitor/meshtastic"
)

var (
	weather libweather.WeatherProvider
	boolStr = map[bool]string{true: "1", false: "0"}
)

type Reporter struct{}

func (r *Reporter) Init(ctx context.Context) {
	if weather != nil {
		panic("Cannot intiialize Reporter twice")
	}

	weather = makeWeatherProvider(appLog)
}

var (
	runtime = libmetric.AutoCommit{Name: "runtime"}

	totalRX       = libmetric.AutoCommit{"total_rx"}
	totalRX_4hops = libmetric.AutoCommit{"total_rx_4hops"}
	totalUnknown  = libmetric.AutoCommit{"total_unknown"}
	totalDecoded  = libmetric.AutoCommit{"total_decoded"}
	senders       = libmetric.AutoCommit{"senders"}

	rssi = libmetric.Sampler{
		Name:     "rssi_p50",
		Function: libmetric.MedianP50,
		MinCount: 100,
		MaxCount: 4000,
	}
	snr = libmetric.Sampler{
		Name:     "snr_p50",
		Function: libmetric.MedianP50,
		MinCount: 100,
		MaxCount: 4000,
	}

	weatherDifficulty = libmetric.Sampler{
		Name:     "weather_difficulty",
		Function: libmetric.Median,
		MinCount: 10,
		MaxCount: 50,
	}
	weatherTempA = libmetric.AutoCommit{"temp_a"}

	groups = []libmetric.Group{
		{Interval: time.Second * 10},
		{Interval: time.Minute},
		{Interval: time.Minute * 5},
		{Interval: time.Minute * 15},
	}
)

func (r *Reporter) Run(ctx context.Context) {
	t0 := groups[0].Ticker()
	t1 := groups[1].Ticker()
	t2 := groups[2].Ticker()
	t3 := groups[3].Ticker()

	commit := func(groupId int) {
		if ok := groups[groupId].Commit(); !ok {
			// todo log
		}
	}

	addRuntime := func(seconds float64) {
		ok := runtime.Add(
			seconds,
			"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
			"pio", myNodeInfo.PioEnv,
			"hw", strconv.Itoa(int(nodeInfo.User.HwModel)),
		)
		if !ok {
			// todo log
		}
	}

	updateWeather := func(groupId int) {
		if weather != nil {
			w, err := weather.GetWeather(ctx)
			if err != nil {
				// todo log
			} else {
				labels := []string{
					"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
					"location", w.Name,
				}
				groups[groupId].Sample(
					&weatherDifficulty, float64(w.RadioDifficulty()),
					labels...,
				)
				groups[groupId].Set(&weatherTempA, float64(w.TempCelsiusFeelsLike), labels...)
			}
		}
	}

	addRuntime(1)

	for {
		select {
		case <-ctx.Done():
			return

		case <-t0.C:
			commit(0)

		case <-t1.C:
			commit(1)

		case <-t2.C:
			updateWeather(2)
			addRuntime(groups[2].Interval.Seconds())
			commit(2)

		case <-t3.C:
			commit(3)
		}
	}
}

func (r *Reporter) HandlePacket(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {

	case *pb.FromRadio_Packet:
		pkt := v.Packet

		if pkt.From == myNodeInfo.MyNodeNum {
			break
		}

		hopsAway := int(meshtastic.HopsAway(pkt))
		if hopsAway > 7 || hopsAway < 0 {
			hopsAway = -1
		}

		labels := []string{
			"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
			"hops", strconv.Itoa(hopsAway),
		}

		isStrong := pkt.RxRssi > -105 && pkt.RxSnr > -5
		labels = append(labels, "strong", boolStr[isStrong])

		groups[0].AddOne(&totalRX, labels...)

		groups[2].Sample(&rssi, float64(pkt.RxRssi), labels...)
		groups[2].Sample(&snr, float64(pkt.RxSnr), labels...)

		d := pkt.GetDecoded()
		if d == nil {
			groups[0].AddOne(&totalUnknown, labels...)
		} else {
			groups[0].AddOne(&totalDecoded, labels...)
		}

		if hopsAway >= 0 && hopsAway <= 4 {
			groups[0].AddOne(&totalRX_4hops, labels...)
		}

		labels = append(labels, "from", fmt.Sprintf("%x", pkt.From))
		groups[1].AddOne(&senders, labels...)
	}
}
