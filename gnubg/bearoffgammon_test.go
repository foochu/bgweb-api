package gnubg

import (
	"reflect"
	"testing"
)

func Test_getBearoffGammonProbs(t *testing.T) {
	type args struct {
		board [6]int
	}
	tests := []struct {
		name string
		args args
		want *_GammonProbs
	}{
		{
			name: "should get 100000",
			args: args{[6]int{1, 0, 0, 0, 0, 0}},
			want: &_GammonProbs{0, 0, 0, 36},
		},
		{
			name: "should get 000001",
			args: args{[6]int{0, 0, 0, 0, 0, 1}},
			want: &_GammonProbs{583, 3203, 15588, 17},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBearoffGammonProbs(tt.args.board); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBearoffGammonProbs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRaceBGprobs(t *testing.T) {
	type args struct {
		board [6]int
	}
	tests := []struct {
		name string
		args args
		want *_RBG
	}{
		{
			name: "should get 100000",
			args: args{[6]int{1, 0, 0, 0, 0, 0}},
			want: &_RBG{36, 0, 0, 0, 0},
		},
		{
			name: "should get 000010",
			args: args{[6]int{0, 0, 0, 0, 1, 0}},
			want: &_RBG{31, 180, 0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRaceBGprobs(tt.args.board); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRaceBGprobs() = %v, want %v", got, tt.want)
			}
		})
	}
}
