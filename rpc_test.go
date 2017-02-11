// Copyright (C) 2016  The GoHBase Authors.  All rights reserved.
// This file is part of GoHBase.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// +build testing,!integration

package gohbase

import (
	"fmt"
	"sync"
	"testing"

	"github.com/aristanetworks/goarista/test"
	"github.com/cznic/b"
	"github.com/golang/mock/gomock"
	"github.com/tsuna/gohbase/hrpc"
	"github.com/tsuna/gohbase/internal/pb"
	"github.com/tsuna/gohbase/region"
	"github.com/tsuna/gohbase/test/mock"
	mockZk "github.com/tsuna/gohbase/test/mock/zk"
	"github.com/tsuna/gohbase/zk"
	"context"
)

func newMockClient(zkClient zk.Client) *client {
	return &client{
		clientType: standardClient,
		regions:    keyRegionCache{regions: b.TreeNew(region.CompareGeneric)},
		clients: clientRegionCache{
			regions: make(map[hrpc.RegionClient][]hrpc.RegionInfo),
		},
		rpcQueueSize:  defaultRPCQueueSize,
		flushInterval: defaultFlushInterval,
		metaRegionInfo: region.NewInfo(
			[]byte("hbase:meta"),
			[]byte("hbase:meta,,1"),
			nil,
			nil),
		zkClient: zkClient,
	}
}

func TestSendRPCSanity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// we expect to ask zookeeper for where metaregion is
	zkClient := mockZk.NewMockClient(ctrl)
	zkClient.EXPECT().LocateResource(zk.Meta).Return(
		"regionserver", uint16(1), nil).MinTimes(1)
	c := newMockClient(zkClient)

	// ask for "theKey" in table "test"
	mockCall := mock.NewMockCall(ctrl)
	mockCall.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockCall.EXPECT().Table().Return([]byte("test")).AnyTimes()
	mockCall.EXPECT().Key().Return([]byte("theKey")).AnyTimes()
	mockCall.EXPECT().SetRegion(gomock.Any()).AnyTimes()
	result := make(chan hrpc.RPCResult, 1)
	// pretend that response is successful
	expMsg := &pb.GetResponse{}
	result <- hrpc.RPCResult{Msg: expMsg}
	mockCall.EXPECT().ResultChan().Return(result).Times(1)
	msg, err := c.sendRPC(mockCall)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if diff := test.Diff(expMsg, msg); diff != "" {
		t.Errorf("Expected: %#v\nReceived: %#v\nDiff:%s",
			expMsg, msg, diff)
	}

	if len(c.clients.regions) != 2 {
		t.Errorf("Expected 2 clients in cache, got %d", len(c.clients.regions))
	}

	// addr -> region name
	expClients := map[string]string{
		"regionserver:1": "hbase:meta,,1",
		"regionserver:2": "test,,1234567890042.56f833d5569a27c7a43fbf547b4924a4.",
	}

	// make sure those are the right clients
	for c, r := range c.clients.regions {
		cAddr := fmt.Sprintf("%s:%d", c.Host(), c.Port())
		name, ok := expClients[cAddr]
		if !ok {
			t.Errorf("Got unexpected client %s:%d in cache", c.Host(), c.Port())
			continue
		}
		if len(r) != 1 {
			t.Errorf("Expected to have only 1 region in cache for client %s:%d",
				c.Host(), c.Port())
			continue
		}
		if string(r[0].Name()) != name {
			t.Errorf("Unexpected name of region %q for client %s:%d, expected %q",
				r[0].Name(), c.Host(), c.Port(), name)
		}
		// check bidirectional mapping, they have to be the same objects
		rc := r[0].Client()
		if c != rc {
			t.Errorf("Invalid bidirectional mapping: forward=%s:%d, backward=%s:%d",
				c.Host(), c.Port(), rc.Host(), rc.Port())
		}
	}

	if c.regions.regions.Len() != 1 {
		// expecting only one region because meta isn't in cache, it's hard-coded
		t.Errorf("Expected 1 regions in cache, got %d", c.regions.regions.Len())
	}
}

func TestReestablishRegionSplit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// we expect to ask zookeeper for where metaregion is
	c := newMockClient(mockZk.NewMockClient(ctrl))

	// inject a fake regionserver client and fake region into cache
	origlReg := region.NewInfo(
		[]byte("test1"),
		[]byte("test1,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		nil,
		nil,
	)
	rc1, err := region.NewClient(context.Background(), "regionserver", 1, region.RegionClient, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	// pretend regionserver:1 has meta table
	c.metaRegionInfo.SetClient(rc1)
	// "test1" is at the moment at regionserver:1
	origlReg.SetClient(rc1)
	// marking unavailable to simulate error
	origlReg.MarkUnavailable()
	c.regions.put(origlReg)
	c.clients.put(rc1, origlReg)
	c.clients.put(rc1, c.metaRegionInfo)

	c.reestablishRegion(origlReg)

	if len(c.clients.regions) != 1 {
		t.Errorf("Expected 1 client in cache, got %d", len(c.clients.regions))
	}

	expRegs := map[string]struct{}{
		"hbase:meta,,1":                                          struct{}{},
		"test1,,1480547738107.825c5c7e480c76b73d6d2bad5d3f7bb8.": struct{}{},
	}

	// make sure those are the right clients
	for rc, rs := range c.clients.regions {
		cAddr := fmt.Sprintf("%s:%d", rc.Host(), rc.Port())
		if cAddr != "regionserver:1" {
			t.Errorf("Got unexpected client %s:%d in cache", rc.Host(), rc.Port())
			break
		}

		// check that we have correct regions in the client
		gotRegs := map[string]struct{}{}
		for _, r := range rs {
			gotRegs[string(r.Name())] = struct{}{}
			// check that regions have correct client
			if r.Client() != rc1 {
				t.Errorf("Invalid bidirectional mapping: forward=%s:%d, backward=%s:%d",
					r.Client().Host(), r.Client().Port(), rc1.Host(), rc1.Port())
			}
		}
		if diff := test.Diff(expRegs, gotRegs); diff != "" {
			t.Errorf("Expected: %#v\nReceived: %#v\nDiff:%s",
				expRegs, gotRegs, diff)
		}

		// check that we still have the same client that we injected
		if rc != rc1 {
			t.Errorf("Invalid bidirectional mapping: forward=%s:%d, backward=%s:%d",
				rc.Host(), rc.Port(), rc1.Host(), rc1.Port())
		}
	}

	if c.regions.regions.Len() != 1 {
		// expecting only one region because meta isn't in cache, it's hard-coded
		t.Errorf("Expected 1 regions in cache, got %d", c.regions.regions.Len())
	}

	// check the we have correct region in regions cache
	newRegIntf, ok := c.regions.regions.Get(
		[]byte("test1,,1480547738107.825c5c7e480c76b73d6d2bad5d3f7bb8."))
	if !ok {
		t.Error("Expected region is not in the cache")
	}

	// check new region is available
	newReg, ok := newRegIntf.(hrpc.RegionInfo)
	if !ok {
		t.Error("Expected hrpc.RegionInfo")
	}
	if newReg.IsUnavailable() {
		t.Error("Expected new region to be available")
	}

	// check old region is available and it's client is empty since we
	// need to release all the waiting callers
	if origlReg.IsUnavailable() {
		t.Error("Expected original region to be available")
	}

	if origlReg.Client() != nil {
		t.Error("Expected original region to have no client")
	}
}

func TestEstablishClientConcurrent(t *testing.T) {
	// test that the same client isn't added when establishing it concurrently
	// if there's a race, this test will only fail sometimes
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// we expect to ask zookeeper for where metaregion is
	c := newMockClient(mockZk.NewMockClient(ctrl))
	// pre-create fake regions to establish
	numRegions := 1000
	regions := make([]hrpc.RegionInfo, numRegions)
	for i := range regions {
		r := region.NewInfo(
			[]byte("test"),
			[]byte(fmt.Sprintf("test,%d,1234567890042.yoloyoloyoloyoloyoloyoloyoloyolo.", i)),
			nil, nil)
		r.MarkUnavailable()
		regions[i] = r
	}

	// all of the regions have the same region client
	var wg sync.WaitGroup
	for _, r := range regions {
		r := r
		wg.Add(1)
		go func() {
			c.establishRegion(r, "regionserver", 1)
			wg.Done()
		}()
	}
	wg.Wait()

	if len(c.clients.regions) != 1 {
		t.Fatalf("Expected to have only 1 client in cache, got %d", len(c.clients.regions))
	}

	for rc, rs := range c.clients.regions {
		if len(rs) != numRegions {
			t.Errorf("Expected to have %d regions, got %d", numRegions, len(rs))
		}
		// check that all regions have the same client set and are available
		for _, r := range regions {
			if r.Client() != rc {
				t.Errorf("Region %q has unexpected client %s:%d",
					r.Name(), r.Client().Host(), r.Client().Port())
			}
			if r.IsUnavailable() {
				t.Errorf("Expected region %s to be available", r.Name())
			}
		}
	}
}
