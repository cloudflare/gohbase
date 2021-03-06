// Copyright (C) 2015  The GoHBase Authors.  All rights reserved.
// This file is part of GoHBase.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package gohbase

import (
	"bytes"
	"reflect"
	"sort"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/tsuna/gohbase/hrpc"
	"github.com/tsuna/gohbase/region"
	mockRegion "github.com/tsuna/gohbase/test/mock/region"
)

func TestMetaCache(t *testing.T) {
	client := newClient("~invalid.quorum~") // We shouldn't connect to ZK.

	reg := client.getRegionFromCache([]byte("test"), []byte("theKey"))
	if reg != nil {
		t.Errorf("Found region %#v even though the cache was empty?!", reg)
	}

	// Inject an entry in the cache.  This entry covers the entire key range.
	wholeTable := region.NewInfo(
		[]byte("test"),
		[]byte("test,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		nil,
		nil,
	)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	regClient := mockRegion.NewMockRegionClient(ctrl)
	regClient.EXPECT().Host().Return("regionserver").AnyTimes()
	regClient.EXPECT().Port().Return(uint16(1)).AnyTimes()

	client.regions.put(wholeTable)
	client.clients.put(regClient, wholeTable)

	reg = client.getRegionFromCache([]byte("test"), []byte("theKey"))
	if !reflect.DeepEqual(reg, wholeTable) {
		t.Errorf("Found region %#v but expected %#v", reg, wholeTable)
	}
	reg = client.getRegionFromCache([]byte("test"), []byte("")) // edge case.
	if !reflect.DeepEqual(reg, wholeTable) {
		t.Errorf("Found region %#v but expected %#v", reg, wholeTable)
	}

	// Clear our client.
	client = newClient("~invalid.quorum~")

	// Inject 3 entries in the cache.
	region1 := region.NewInfo(
		[]byte("test"),
		[]byte("test,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte(""),
		[]byte("foo"),
	)
	client.regions.put(region1)
	client.clients.put(regClient, region1)

	region2 := region.NewInfo(
		[]byte("test"),
		[]byte("test,foo,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte("foo"),
		[]byte("gohbase"),
	)
	client.regions.put(region2)
	client.clients.put(regClient, region2)

	region3 := region.NewInfo(
		[]byte("test"),
		[]byte("test,gohbase,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte("gohbase"),
		[]byte(""),
	)
	client.regions.put(region3)
	client.clients.put(regClient, region3)

	testcases := []struct {
		key string
		reg hrpc.RegionInfo
	}{
		{key: "theKey", reg: region3},
		{key: "", reg: region1},
		{key: "bar", reg: region1},
		{key: "fon\xFF", reg: region1},
		{key: "foo", reg: region2},
		{key: "foo\x00", reg: region2},
		{key: "gohbase", reg: region3},
	}
	for i, testcase := range testcases {
		reg = client.getRegionFromCache([]byte("test"), []byte(testcase.key))
		if !reflect.DeepEqual(reg, testcase.reg) {
			t.Errorf("[#%d] Found region %#v but expected %#v", i, reg, testcase.reg)
		}
	}

	// Change the last region (maybe it got split).
	region3 = region.NewInfo(
		[]byte("test"),
		[]byte("test,gohbase,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		nil,
		[]byte("zab"),
	)
	client.regions.put(region3)
	client.clients.put(regClient, region3)

	reg = client.getRegionFromCache([]byte("test"), []byte("theKey"))
	if !reflect.DeepEqual(reg, region3) {
		t.Errorf("Found region %#v but expected %#v", reg, region3)
	}
	reg = client.getRegionFromCache([]byte("test"), []byte("zoo"))
	if reg != nil {
		t.Errorf("Shouldn't have found any region yet found %#v", reg)
	}
}

type regionNames [][]byte

func (a regionNames) Len() int           { return len(a) }
func (a regionNames) Less(i, j int) bool { return bytes.Compare(a[i], a[j]) < 0 }
func (a regionNames) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func TestMetaCacheGetOverlaps(t *testing.T) {
	region1 := region.NewInfo(
		[]byte("test"),
		[]byte("test,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte(""),
		[]byte("foo"),
	)

	regionA := region.NewInfo(
		[]byte("hello"),
		[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte(""),
		[]byte("foo"),
	)

	regionB := region.NewInfo(
		[]byte("hello"),
		[]byte("hello,foo,987654321042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte("foo"),
		[]byte("fox"),
	)

	regionC := region.NewInfo(
		[]byte("hello"),
		[]byte("hello,fox,987654321042.56f833d5569a27c7a43fbf547b4924a4."),
		[]byte("fox"),
		[]byte("yolo"),
	)

	regionWhole := region.NewInfo(
		[]byte("hello"),
		[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
		nil,
		nil,
	)

	regionTests := []struct {
		cachedRegions []hrpc.RegionInfo
		newRegion     hrpc.RegionInfo
		expected      []hrpc.RegionInfo
	}{
		{[]hrpc.RegionInfo{}, region1, []hrpc.RegionInfo{}},               // empty cache
		{[]hrpc.RegionInfo{region1}, region1, []hrpc.RegionInfo{region1}}, // with itself
		{ // different table
			[]hrpc.RegionInfo{region1},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte(""),
				[]byte("fake"),
			),
			[]hrpc.RegionInfo{},
		},
		{ // overlaps with both
			[]hrpc.RegionInfo{regionA, regionB},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,bar,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte("bar"),
				[]byte("fop"),
			),
			[]hrpc.RegionInfo{regionA, regionB},
		},
		{ // overlaps with both, key start == old one
			[]hrpc.RegionInfo{regionA, regionB},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte(""),
				[]byte("yolo"),
			),
			[]hrpc.RegionInfo{regionA, regionB},
		},
		{ // overlaps with second
			[]hrpc.RegionInfo{regionA, regionB},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,fop,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte("fop"),
				[]byte("yolo"),
			),
			[]hrpc.RegionInfo{regionB},
		},
		{ // overlaps with first, new key start == old one
			[]hrpc.RegionInfo{regionA, regionB},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte(""),
				[]byte("abc"),
			),
			[]hrpc.RegionInfo{regionA},
		},
		{ // doesn't overlap, is between existing
			[]hrpc.RegionInfo{regionA, regionC},
			regionB,
			[]hrpc.RegionInfo{},
		},
		{ // without bounds in cache, replaced by region with both bounds
			[]hrpc.RegionInfo{regionWhole},
			regionB,
			[]hrpc.RegionInfo{regionWhole},
		},
		{ // without bounds in cache, replaced by the empty stop key only
			[]hrpc.RegionInfo{regionWhole},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				[]byte("yolo"),
				nil,
			),
			[]hrpc.RegionInfo{regionWhole},
		},
		{ // without bounds in cache, replaced by the empty start key only
			[]hrpc.RegionInfo{regionWhole},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.56f833d5569a27c7a43fbf547b4924a4."),
				nil,
				[]byte("yolo"),
			),
			[]hrpc.RegionInfo{regionWhole},
		},
		{ // regions with bounds in cache, replaced by without bounds
			[]hrpc.RegionInfo{regionB, regionC},
			regionWhole,
			[]hrpc.RegionInfo{regionB, regionC},
		},
		{ // without bounds in cache, replaced by without bounds
			[]hrpc.RegionInfo{regionWhole},
			region.NewInfo(
				[]byte("hello"),
				[]byte("hello,,1234567890042.yoloyoloyoloyoloyoloyoloyoloyolo."),
				nil,
				nil,
			),
			[]hrpc.RegionInfo{regionWhole},
		},
	}

	client := newClient("~invalid.quorum~") // fake client
	for i, tt := range regionTests {
		client.regions.regions.Clear()
		// set up initial cache
		for _, region := range tt.cachedRegions {
			client.regions.put(region)
		}

		expectedNames := make(regionNames, len(tt.expected))
		for i, r := range tt.expected {
			expectedNames[i] = r.Name()
		}
		os := client.regions.getOverlaps(tt.newRegion)
		osNames := make(regionNames, len(os))
		for i, o := range os {
			osNames[i] = o.Name()
		}
		sort.Sort(expectedNames)
		sort.Sort(osNames)
		if !reflect.DeepEqual(expectedNames, osNames) {
			t.Errorf("=== TestMetaCacheGetOverlaps #%d: Expected overlaps %q, found %q", i+1,
				expectedNames, osNames)
		}
	}
}

func TestClientCachePut(t *testing.T) {
	client := newClient("~invalid.quorum~")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	regClient := mockRegion.NewMockRegionClient(ctrl)
	regClient.EXPECT().Host().Return("regionserver").AnyTimes()
	regClient.EXPECT().Port().Return(uint16(1)).AnyTimes()

	existing := client.clients.put(regClient, region.NewInfo(
		[]byte("test"),
		[]byte("test,,1234567890042.yoloyoloyoloyoloyoloyoloyoloyolo."),
		nil, nil))

	if existing != regClient {
		t.Errorf("Unexpected regClient from put: %v", existing)
	}

	if len(client.clients.regions) != 1 {
		t.Errorf("Expected 1 client in cache, got %d", len(client.clients.regions))
	}

	if len(client.clients.regions[regClient]) != 1 {
		t.Errorf("Expected 1 region for client in cache, got %d",
			len(client.clients.regions[regClient]))
	}

	// try putting client with the same host port
	regClient2 := mockRegion.NewMockRegionClient(ctrl)
	regClient2.EXPECT().Host().Return("regionserver").AnyTimes()
	regClient2.EXPECT().Port().Return(uint16(1)).AnyTimes()

	// but put a different region
	existing = client.clients.put(regClient, region.NewInfo(
		[]byte("yolo"),
		[]byte("yolo,,1234567890042.yoloyoloyoloyoloyoloyoloyoloyolo."),
		nil, nil))

	if existing != regClient {
		t.Errorf("Unexpected regClient from put: %v", existing)
	}

	// nothing should have changed in clients cache
	if len(client.clients.regions) != 1 {
		t.Errorf("Expected 1 client in cache, got %d", len(client.clients.regions))
	}

	if len(client.clients.regions[regClient]) != 2 {
		t.Errorf("Expected 2 regions for client in cache, got %d",
			len(client.clients.regions[regClient]))
	}
}
