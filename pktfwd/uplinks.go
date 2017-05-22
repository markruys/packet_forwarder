// Copyright © 2017 The Things Network. Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pktfwd

import (
	"errors"

	"github.com/TheThingsNetwork/packet_forwarder/wrapper"
	"github.com/TheThingsNetwork/ttn/api/gateway"
	"github.com/TheThingsNetwork/ttn/api/protocol"
	"github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	"github.com/TheThingsNetwork/ttn/api/router"
)

func acceptedCRC(p wrapper.Packet) bool {
	// XX: Should retrieve the CRC configuration from the account server.
	if p.Status == wrapper.StatusCRCOK || p.Status == wrapper.StatusNOCRC {
		return true
	}
	return false
}

func initGatewayMetadata(gatewayID string, packet wrapper.Packet) gateway.RxMetadata {
	var gateway = gateway.RxMetadata{
		GatewayId: gatewayID,
		RfChain:   uint32(packet.RFChain),
		Channel:   uint32(packet.IFChain),
		Frequency: uint64(packet.Freq),
		Rssi:      packet.RSSI,
		Snr:       packet.SNR,
		Timestamp: packet.CountUS,
		Time:      packet.Time,
		Gps:       packet.Gps,
	}
	return gateway
}

func newLoRaMetadata(packet wrapper.Packet) (lorawan.Metadata, error) {
	var datarate, bandwidth, coderate string
	var err error

	var p = lorawan.Metadata{
		Modulation: lorawan.Modulation_LORA,
	}

	if datarate, err = packet.DatarateString(); err != nil {
		return p, err
	}
	if bandwidth, err = packet.BandwidthString(); err != nil {
		return p, err
	}
	p.DataRate = datarate + bandwidth

	if coderate, err = packet.CoderateString(); err != nil {
		return p, err
	}
	p.CodingRate = coderate

	return p, nil
}

func newFSKMetadata(packet wrapper.Packet) lorawan.Metadata {
	var p = lorawan.Metadata{
		Modulation: lorawan.Modulation_FSK,
	}
	p.BitRate = packet.Datarate
	return p
}

func initLoRaData(packet wrapper.Packet) (lorawan.Metadata, error) {
	var loRaData lorawan.Metadata
	if packet.Modulation == wrapper.ModulationLoRa {
		var err error
		loRaData, err = newLoRaMetadata(packet)
		if err != nil {
			return loRaData, err
		}
	} else if packet.Modulation == wrapper.ModulationFSK {
		loRaData = newFSKMetadata(packet)
	} else {
		return loRaData, errors.New("Received packet with unknown modulation")
	}

	return loRaData, nil
}

func createUplinkMessage(gatewayID string, packet wrapper.Packet) (router.UplinkMessage, error) {
	var uplink router.UplinkMessage

	gateway := initGatewayMetadata(gatewayID, packet)
	loraData, err := initLoRaData(packet)
	if err != nil {
		return uplink, err
	}
	var data = protocol.RxMetadata{
		Protocol: &protocol.RxMetadata_Lorawan{
			Lorawan: &loraData,
		},
	}

	uplink = router.UplinkMessage{
		ProtocolMetadata: &data,
		GatewayMetadata:  &gateway,
		Payload:          packet.Payload,
	}

	return uplink, nil
}

func wrapUplinkPayload(packets []wrapper.Packet, ignoreCrc bool, gatewayID string) ([]router.UplinkMessage, error) {
	var messages = make([]router.UplinkMessage, 0, wrapper.NbMaxPackets)
	// Iterating through every packet:
	for _, inspectedPacket := range packets {
		// First, we'll check the CRC is conform to the packets the gateway is configured to transmit
		if !ignoreCrc && !acceptedCRC(inspectedPacket) {
			continue
		}

		// Creating and filling the uplink message
		message, err := createUplinkMessage(gatewayID, inspectedPacket)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}
