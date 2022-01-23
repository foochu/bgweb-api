package gnubg

import "testing"

func Test_cacheLookup(t *testing.T) {
	// create cache
	var pc _EvalCache
	cacheCreate(&pc, 19)
	// add item
	cacheAdd(&pc, &_CacheNodeDetail{
		key: _PositionKey{
			data: [7]int{810549248, 372680, 16842752, 810549248, 327680, 536870912, 0},
		},
		nEvalContext: 0,
	}, 9)
	// run tests
	type args struct {
		pc        *_EvalCache
		e         *_CacheNodeDetail
		arOut     *[_NUM_OUTPUTS]float32
		arCubeful *float32
	}
	tests := []struct {
		name    string
		args    args
		wantHit bool
		wantL   _HashKey
	}{
		{
			name: "should lookup - cache hit",
			args: args{
				pc: &pc,
				e: &_CacheNodeDetail{
					key: _PositionKey{
						data: [7]int{810549248, 372680, 16842752, 810549248, 327680, 536870912, 0},
					},
					nEvalContext: 0,
				},
				arOut:     &[5]float32{},
				arCubeful: nil,
			},
			wantHit: true,
			wantL:   9,
		},
		{
			name: "should lookup - cache miss",
			args: args{
				pc: &pc,
				e: &_CacheNodeDetail{
					key: _PositionKey{
						data: [7]int{810549248, 372680, 16842752, 810549248, 327680, 536870912, 1},
					},
					nEvalContext: 0,
				},
				arOut:     &[5]float32{},
				arCubeful: nil,
			},
			wantHit: false,
			wantL:   15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHit, gotL := cacheLookup(tt.args.pc, tt.args.e, tt.args.arOut, tt.args.arCubeful)
			if gotHit != tt.wantHit {
				t.Errorf("cacheLookup() gotHit = %v, want %v", gotHit, tt.wantHit)
			}
			if gotL != tt.wantL {
				t.Errorf("cacheLookup() gotL = %v, want %v", gotL, tt.wantL)
			}
		})
	}
	// destroy cache
	cacheDestroy(&pc)
}
