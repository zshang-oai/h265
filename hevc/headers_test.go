package hevc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type streamInfo struct {
	width, height uint32
	chroma        uint32
	bitDepth      uint8
	profile       uint8
	level         uint8
}

// Reference values from ffprobe.
var streams = map[string]streamInfo{
	"1080p.h265":                           {1920, 1080, 1, 8, 1, 120},
	"10bit_128x128.h265":                   {128, 128, 1, 10, 2, 30},
	"angular_grad.h265":                    {16, 16, 1, 8, 3, 30},
	"angular.h265":                         {16, 16, 1, 8, 3, 30},
	"aq_intra_64.h265":                     {64, 64, 1, 8, 3, 30},
	"aq_p_320x240.h265":                    {320, 240, 1, 8, 1, 60},
	"bframes3_128x128.h265":                {128, 128, 1, 8, 1, 30},
	"constrained_intra_128x128.h265":       {128, 128, 1, 8, 1, 30},
	"ctu64_128x128.h265":                   {128, 128, 1, 8, 1, 30},
	"ctu64_64_nosis.h265":                  {64, 64, 1, 8, 4, 30},
	"ctu64_aq.h265":                        {320, 240, 1, 8, 1, 60},
	"ctu64_noqp_nosao_320x240.h265":        {320, 240, 1, 8, 1, 60},
	"ctu64_wpp.h265":                       {320, 240, 1, 8, 1, 60},
	"deblock_grad.h265":                    {32, 32, 1, 8, 3, 30},
	"deblock_sao_320x240.h265":             {320, 240, 1, 8, 1, 60},
	"deblock_sao_nodeblock.h265":           {320, 240, 1, 8, 1, 60},
	"dep_slices.h265":                      {128, 128, 1, 8, 1, 186},
	"flat64.h265":                          {64, 64, 1, 8, 3, 30},
	"fuzz_bit_depth_change.h265":           {320, 240, 1, 8, 1, 60},
	"fuzz_mvd_overflow.h265":               {384, 216, 1, 8, 1, 60},
	"grad64.h265":                          {64, 64, 1, 8, 3, 30},
	"inter_b.h265":                         {16, 16, 1, 8, 1, 30},
	"inter_p.h265":                         {16, 16, 1, 8, 1, 30},
	"motion_320x240.h265":                  {320, 240, 1, 8, 1, 60},
	"multi_ctu.h265":                       {32, 32, 1, 8, 3, 30},
	"multi_slice.h265":                     {64, 64, 1, 8, 3, 30},
	"multi_slice_sao_deblock_256x256.h265": {256, 256, 1, 8, 1, 60},
	"no_filter_across_slices_256x256.h265": {256, 256, 1, 8, 1, 186},
	"pcm.h265":                             {16, 16, 1, 8, 3, 30},
	"pcm_hm_64x64.h265":                    {64, 64, 1, 8, 0, 0},
	"qp_small.h265":                        {64, 64, 1, 8, 3, 30},
	"ramp64.h265":                          {64, 64, 1, 8, 3, 30},
	"realworld_320x240.h265":               {320, 240, 1, 8, 1, 60},
	"realworld_720p.h265":                  {1280, 720, 1, 8, 1, 93},
	"sao.h265":                             {16, 16, 1, 8, 3, 30},
	"scaling_list.h265":                    {16, 16, 1, 8, 3, 30},
	"signhide.h265":                        {16, 16, 1, 8, 3, 30},
	"signhide_scaling_320x240.h265":        {320, 240, 1, 8, 1, 60},
	"tiles.h265":                           {256, 256, 1, 8, 1, 186},
	"tiny_intra.h265":                      {16, 16, 1, 8, 3, 30},
	"transquant_bypass_64x64.h265":         {64, 64, 1, 8, 1, 255},
	"tskip_128x128.h265":                   {128, 128, 1, 8, 1, 30},
	"tu32.h265":                            {32, 32, 1, 8, 3, 30},
	"tu32_ionly.h265":                      {32, 32, 1, 8, 4, 30},
	"tu32_nowpp.h265":                      {32, 32, 1, 8, 1, 30},
	"tu32_test.h265":                       {32, 32, 1, 8, 1, 30},
	"tu8.h265":                             {16, 16, 1, 8, 3, 30},
	"tu_inter4x4_motion.h265":              {128, 128, 1, 8, 1, 30},
	"wpp_ctu16.h265":                       {384, 216, 1, 8, 1, 60},
}

func testdataFiles(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("testdata", "*.h265"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Skip("no testdata streams")
	}

	return files
}

func TestParameterSets(t *testing.T) {
	for _, f := range testdataFiles(t) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		name := filepath.Base(f)

		var (
			nVPS, nSPS, nPPS int
			active           *sps
		)

		for _, nal := range SplitAnnexB(data) {
			switch nal.Type {
			case NALVPS:
				if _, err := parseVPS(nal.RBSP); err != nil {
					t.Errorf("%s: VPS: %v", name, err)

					continue
				}

				nVPS++

			case NALSPS:
				s, err := parseSPS(nal.RBSP)
				if errors.Is(err, ErrUnsupported) {
					t.Skipf("%s: %v", name, err)
				}

				if err != nil {
					t.Errorf("%s: SPS: %v", name, err)

					continue
				}

				nSPS++
				active = s

			case NALPPS:
				p, err := parsePPS(nal.RBSP)
				if err != nil {
					t.Errorf("%s: PPS: %v", name, err)

					continue
				}

				nPPS++

				if active == nil {
					continue
				}

				if err := p.resolveTileGeometry(active); err != nil {
					t.Errorf("%s: tile geometry: %v", name, err)

					continue
				}

				var sum uint32
				for _, w := range p.colWidthsInCtbs {
					sum += w
				}

				if sum != active.picWidthInCtbs {
					t.Errorf("%s: tile columns sum to %d, want %d", name, sum, active.picWidthInCtbs)
				}

				sum = 0
				for _, h := range p.rowHeightsInCtbs {
					sum += h
				}

				if sum != active.picHeightInCtbs {
					t.Errorf("%s: tile rows sum to %d, want %d", name, sum, active.picHeightInCtbs)
				}
			}
		}

		if nVPS == 0 || nSPS == 0 || nPPS == 0 {
			t.Errorf("%s: got %d VPS, %d SPS, %d PPS", name, nVPS, nSPS, nPPS)
		}

		want, ok := streams[name]
		if !ok || active == nil {
			continue
		}

		got := streamInfo{
			width:    active.croppedWidth(),
			height:   active.croppedHeight(),
			chroma:   active.chromaFormatIDC,
			bitDepth: active.bitDepthLuma,
			profile:  active.ptl.profileIDC,
			level:    active.ptl.levelIDC,
		}

		if got != want {
			t.Errorf("%s: got %+v, want %+v", name, got, want)
		}
	}
}

func TestSPSOrderingLimits(t *testing.T) {
	for _, tt := range []struct {
		name          string
		level         uint8
		width, height uint32
		profile       uint8
		compat        uint32
		highestOnly   bool
		unspecified   bool
		layers        [][3]uint32 // DPB minus1, reorder depth, latency plus1
		want          error
	}{
		{name: "immediate output", layers: [][3]uint32{{0, 0, 0}}},
		{name: "largest DPB", layers: [][3]uint32{{15, 15, 0}}},
		{name: "increasing layers", layers: [][3]uint32{{1, 0, 0}, {3, 2, 0}}},
		{name: "inferred lower layer", highestOnly: true, layers: [][3]uint32{{3, 2, 0}, {3, 2, 0}}},
		{name: "largest latency", layers: [][3]uint32{{1, 0, 1<<32 - 2}}},
		{name: "too many pictures", layers: [][3]uint32{{16, 0, 0}}, want: ErrInvalid},
		{name: "unbounded level within budget", level: 255, layers: [][3]uint32{{15, 15, 0}}},
		{name: "unbounded level storage", level: 255, layers: [][3]uint32{{16, 0, 0}}, want: ErrUnsupported},
		{name: "reorder exceeds capacity", layers: [][3]uint32{{0, 1, 0}}, want: ErrInvalid},
		{name: "decreasing capacity", layers: [][3]uint32{{3, 0, 0}, {1, 0, 0}}, want: ErrInvalid},
		{name: "decreasing reorder depth", layers: [][3]uint32{{3, 2, 0}, {3, 1, 0}}, want: ErrInvalid},
		// A.4.2 changes the allowance at one quarter, one half and three
		// quarters of MaxLumaPs. These coded dimensions straddle each edge.
		{name: "quarter", width: 96, height: 96, layers: [][3]uint32{{15, 15, 0}}},
		{name: "above quarter", width: 112, height: 96, layers: [][3]uint32{{12, 0, 0}}, want: ErrInvalid},
		{name: "half", width: 192, height: 96, layers: [][3]uint32{{11, 11, 0}}},
		{name: "above half", width: 208, height: 96, layers: [][3]uint32{{8, 0, 0}}, want: ErrInvalid},
		{name: "three quarters", width: 288, height: 96, layers: [][3]uint32{{7, 7, 0}}},
		{name: "above three quarters", width: 304, height: 96, layers: [][3]uint32{{6, 0, 0}}, want: ErrInvalid},
		{name: "full level picture", width: 192, height: 192, layers: [][3]uint32{{5, 5, 0}}},
		{name: "picture exceeds level", width: 208, height: 192, layers: [][3]uint32{{0, 0, 0}}, want: ErrInvalid},
		{name: "1080p level 4.1", level: 123, width: 1920, height: 1088, layers: [][3]uint32{{5, 5, 0}}},
		{name: "1080p level 4.1 over capacity", level: 123, width: 1920, height: 1088, layers: [][3]uint32{{6, 0, 0}}, want: ErrInvalid},
		{name: "8K level 6.2", level: 186, width: 7680, height: 4320, layers: [][3]uint32{{6, 0, 0}}, want: ErrInvalid},
		{name: "8K level 6.3", level: 189, width: 7680, height: 4320, layers: [][3]uint32{{11, 11, 0}}},
		{name: "8K level 7.2", level: 216, width: 7680, height: 4320, layers: [][3]uint32{{15, 15, 0}}},
		{name: "unknown level", level: 217, layers: [][3]uint32{{0, 0, 0}}, want: ErrUnsupported},
		// Existing x265 still streams overdeclare storage; keep accepting
		// them within the general video-level budget.
		{name: "still storage compatibility", profile: 3, layers: [][3]uint32{{4, 2, 0}}},
		{name: "still exceeds general budget", profile: 3, layers: [][3]uint32{{16, 0, 0}}, want: ErrInvalid},
		{name: "legacy unspecified PTL", unspecified: true, layers: [][3]uint32{{15, 15, 0}}},
		{name: "legacy storage budget", unspecified: true, layers: [][3]uint32{{16, 0, 0}}, want: ErrUnsupported},
		{name: "SCC base capacity", profile: 9, width: 192, height: 192, layers: [][3]uint32{{6, 6, 0}}},
		{name: "SCC Main compatibility", profile: 9, compat: 1 << (31 - 1), width: 192, height: 192, layers: [][3]uint32{{6, 0, 0}}, want: ErrInvalid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Build a complete small SPS so rejection cannot be explained by
			// a missing field or truncated trailing bits after the ordering data.
			var w putBits
			w.bits(0, 4)
			w.bits(uint64(len(tt.layers)-1), 3)
			w.bit(1)
			profile, level := tt.profile, tt.level
			if profile == 0 && !tt.unspecified {
				profile = 1
			}
			if level == 0 && !tt.unspecified {
				level = 30
			}
			w.bits(0, 3) // profile space and tier
			w.bits(uint64(profile), 5)
			w.bits(uint64(tt.compat), 32)
			w.bits(0b1011, 4) // progressive, non-packed, frame-only
			w.bits(0, 44)     // profile constraints
			w.bits(uint64(level), 8)
			if len(tt.layers) > 1 {
				w.bits(0, 16) // sub-layer profile/level flags and reserved bits
			}
			w.ue(0) // SPS ID
			w.ue(1) // 4:2:0
			width, height := tt.width, tt.height
			if width == 0 {
				width, height = 64, 64
			}
			w.ue(width)
			w.ue(height)
			w.bit(0) // no conformance window
			w.ue(0)  // eight-bit luma
			w.ue(0)  // eight-bit chroma
			w.ue(4)  // eight-bit POC LSB
			w.bit(boolToBit(!tt.highestOnly))
			layers := tt.layers
			if tt.highestOnly {
				layers = layers[len(layers)-1:]
			}
			for _, layer := range layers {
				for _, v := range layer {
					w.ue(v)
				}
			}
			w.ue(1)      // minimum coding block: 16
			w.ue(0)      // coding tree block: 16
			w.ue(0)      // minimum transform: 4
			w.ue(2)      // maximum transform: 16
			w.ue(0)      // inter transform hierarchy
			w.ue(0)      // intra transform hierarchy
			w.bits(0, 4) // scaling lists, AMP, SAO, PCM disabled
			w.ue(0)      // no short-term reference sets
			w.bits(0, 5) // long-term refs, temporal MVP, smoothing, VUI, extensions
			w.rbspTrailingBits()

			s, err := parseSPS(w.bytes())
			if !errors.Is(err, tt.want) {
				t.Fatalf("parseSPS: got %v, want %v", err, tt.want)
			}
			if tt.want == nil {
				got := [3]uint32{s.maxDecPicBuffering, s.maxNumReorderPics, s.maxLatencyIncrease}
				if want := tt.layers[len(tt.layers)-1]; got != want {
					t.Fatalf("highest sub-layer limits: got %v, want %v", got, want)
				}
				for i, layer := range tt.layers {
					if s.maxDecPicBufferingByLayer[i] != layer[0] {
						t.Fatalf("sub-layer %d capacity: got %d, want %d", i, s.maxDecPicBufferingByLayer[i]+1, layer[0]+1)
					}
				}
			}
		})
	}
}

func TestDefaultScalingList(t *testing.T) {
	sl := defaultScalingList()

	for m := range sl.sl[0] {
		for i := range 16 {
			if sl.sl[0][m][i] != 16 {
				t.Fatalf("4x4 [%d][%d] = %d", m, i, sl.sl[0][m][i])
			}
		}
	}

	for s := 1; s < maxScalingListSizes; s++ {
		for m := range sl.sl[s] {
			want := defaultScalingListIntra
			if m >= 3 {
				want = defaultScalingListInter
			}

			if sl.sl[s][m] != want {
				t.Fatalf("size %d matrix %d mismatch", s, m)
			}
		}
	}

	for s := range sl.dc {
		for m := range sl.dc[s] {
			if sl.dc[s][m] != 16 {
				t.Fatalf("dc [%d][%d] = %d", s, m, sl.dc[s][m])
			}
		}
	}
}

// bitPut writes the Exp-Golomb forms of 9.2 so a syntax structure can be
// assembled without a stream to carry it.
type bitPut struct {
	b   []byte
	n   int
	acc byte
}

func (p *bitPut) bit(v uint32) {
	p.acc = p.acc<<1 | byte(v&1)
	p.n++

	if p.n == 8 {
		p.b = append(p.b, p.acc)
		p.acc, p.n = 0, 0
	}
}

func (p *bitPut) ue(v uint32) {
	n := v + 1

	k := 0
	for n>>(k+1) != 0 {
		k++
	}

	for range k {
		p.bit(0)
	}

	for i := k; i >= 0; i-- {
		p.bit(n >> uint(i))
	}
}

func (p *bitPut) se(v int32) {
	if v > 0 {
		p.ue(uint32(2*v - 1))

		return
	}

	p.ue(uint32(-2 * v))
}

func (p *bitPut) bytes() []byte {
	for p.n != 0 {
		p.bit(0)
	}

	return append(p.b, 0, 0, 0, 0)
}

// TestPredWeightTableOffsetRange covers 7.4.7.3's WpOffsetHalfRangeC, which is
// the sample depth under high_precision_offsets_enabled_flag and 128 otherwise.
func TestPredWeightTableOffsetRange(t *testing.T) {
	for _, tt := range []struct {
		name          string
		highPrecision bool
		bitDepth      uint8
		want          [2]int16
	}{
		{"eight bit", false, 8, [2]int16{127, -128}},
		{"ten bit high precision", true, 10, [2]int16{300, -300}},
		{"ten bit", false, 10, [2]int16{127, -128}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var p bitPut

			p.ue(0)
			p.se(0)
			p.bit(0)
			p.bit(1)
			p.se(0)
			p.se(300)
			p.se(0)
			p.se(-300)

			var g getBits

			g.init(p.bytes())

			s := &sps{
				chromaFormatIDC:      1,
				bitDepthChroma:       tt.bitDepth,
				highPrecisionOffsets: tt.highPrecision,
			}

			h := &sliceHeader{sliceType: sliceP, numRefIdxL0Active: 1}

			if err := parsePredWeightTable(&g, h, s); err != nil {
				t.Fatal(err)
			}

			if got := h.weights.chromaOffset[0][0]; got != tt.want {
				t.Fatalf("chromaOffset = %v, want %v", got, tt.want)
			}
		})
	}
}

// colorDesc is what the video usability information says about the samples.
type colorDesc struct {
	primaries, transfer, matrix uint16
	fullRange                   bool
}

// TestParseVUIColorDescription covers E.2.1's video_signal_type, which is where
// a sequence declares the range and matrix its samples are in. A container with
// no description of its own falls back to it.
func TestParseVUIColorDescription(t *testing.T) {
	for _, tt := range []struct {
		name              string
		signal, desc      bool
		full              bool
		prim, trc, matrix uint32
		want              colorDesc
	}{
		{
			name: "absent", want: colorDesc{2, 2, 2, false},
		},
		{
			name: "range only", signal: true, full: true,
			want: colorDesc{2, 2, 2, true},
		},
		{
			name: "full description", signal: true, desc: true, full: true,
			prim: 1, trc: 13, matrix: 6,
			want: colorDesc{1, 13, 6, true},
		},
		{
			name: "limited range", signal: true, desc: true,
			prim: 9, trc: 16, matrix: 9,
			want: colorDesc{9, 16, 9, false},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var p bitPut

			p.bit(0) // aspect_ratio_info_present_flag
			p.bit(0) // overscan_info_present_flag

			if !tt.signal {
				p.bit(0)
			} else {
				p.bit(1)
				p.bit(0)
				p.bit(0)
				p.bit(0) // video_format

				p.bit(b2u(tt.full))

				if !tt.desc {
					p.bit(0)
				} else {
					p.bit(1)

					for _, v := range []uint32{tt.prim, tt.trc, tt.matrix} {
						for i := 7; i >= 0; i-- {
							p.bit(v >> uint(i))
						}
					}
				}
			}

			// chroma_loc, neutral_chroma, field_seq, frame_field, display
			// window, timing and bitstream restriction, all absent.
			for range 7 {
				p.bit(0)
			}

			var c getBits

			c.init(p.bytes())

			s := &sps{colourPrimaries: 2, transferChar: 2, matrixCoeffs: 2}

			if err := parseVUI(&c, s, 0); err != nil {
				t.Fatal(err)
			}

			got := colorDesc{s.colourPrimaries, s.transferChar, s.matrixCoeffs, s.fullRange}

			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func b2u(v bool) uint32 {
	if v {
		return 1
	}

	return 0
}

func TestWriteParameterSets(t *testing.T) {
	h := encoderHeaders{width: 320, height: 240, levelIDC: 60, pcm: true, signDataHidingEnabled: true,
		chromaFormat: 1, subWidthC: 2, subHeightC: 2}

	v, err := parseVPS(h.vps())
	if err != nil {
		t.Fatalf("VPS: %v", err)
	}

	if v.id != 0 || v.ptl.profileIDC != 1 || v.ptl.levelIDC != h.levelIDC || v.ptl.tierFlag {
		t.Fatalf("VPS = %+v", v)
	}

	s, err := parseSPS(h.sps())
	if err != nil {
		t.Fatalf("SPS: %v", err)
	}

	if s.vpsID != 0 || s.picWidthInLumaSamples != uint32(h.width) ||
		s.picHeightInLumaSamples != uint32(h.height) || s.chromaFormatIDC != 1 ||
		s.bitDepthLuma != 8 || s.bitDepthChroma != 8 || !s.pcmEnabled ||
		s.log2MinPcmCbSize != 4 || s.log2MaxPcmCbSize != 4 ||
		!s.pcmLoopFilterDisabled {
		t.Fatalf("SPS = %+v", s)
	}

	// ISO/IEC 14496-15 has a container repeat these twelve bytes, so they have
	// to be the ones the sequence parameter set itself carries.
	ptl, ok := ProfileTierLevel(h.sps())
	if !ok {
		t.Fatal("no profile_tier_level")
	}

	if len(ptl) != 12 || ptl[0]>>5 != 0 || ptl[0]&0x1f != 1 || ptl[11] != h.levelIDC {
		t.Fatalf("profile_tier_level = %x", ptl)
	}

	if _, ok := ProfileTierLevel(h.sps()[:12]); ok {
		t.Fatal("short SPS accepted")
	}

	p, err := parsePPS(h.pps())
	if err != nil {
		t.Fatalf("PPS: %v", err)
	}

	if p.id != 0 || p.spsID != 0 || !p.signDataHidingEnabled || !p.deblockingControlPresen ||
		p.deblockingDisabled || p.tilesEnabled || p.entropyCodingSync ||
		!p.loopFilterAcrossSlices {
		t.Fatalf("PPS = %+v", p)
	}

	// The flags either side of entropy_coding_sync are written positionally,
	// so each one is read back with the others held where they were.
	h.wavefront = true

	p, err = parsePPS(h.pps())
	if err != nil {
		t.Fatalf("wavefront PPS: %v", err)
	}

	if !p.entropyCodingSync || p.tilesEnabled || !p.loopFilterAcrossSlices ||
		!p.deblockingControlPresen || p.deblockingDisabled || !p.signDataHidingEnabled ||
		p.betaOffsetDiv2 != 0 || p.tcOffsetDiv2 != 0 {
		t.Fatalf("wavefront PPS = %+v", p)
	}

	// pps_deblocking_filter_disabled_flag governs whether the two offsets
	// behind it are there at all, so the flags after them move with it.
	h.deblockingDisabled = true

	p, err = parsePPS(h.pps())
	if err != nil {
		t.Fatalf("no deblocking PPS: %v", err)
	}

	if !p.deblockingDisabled || !p.entropyCodingSync || !p.loopFilterAcrossSlices ||
		!p.deblockingControlPresen || !p.signDataHidingEnabled ||
		p.scalingListPresent || p.listsModificationPresent {
		t.Fatalf("no deblocking PPS = %+v", p)
	}
}
