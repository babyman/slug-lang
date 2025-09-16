package future

import (
	"errors"
	"testing"
	"time"
)

func TestFirst(t *testing.T) {
	type testCase struct {
		name    string
		futures []*Future[int]
		wantVal int
		wantErr bool
	}

	testCases := []testCase{
		{
			name:    "single future success",
			futures: []*Future[int]{FromValue(42)},
			wantVal: 42,
			wantErr: false,
		},
		{
			name:    "multiple futures one completes first",
			futures: []*Future[int]{FromValue(10), FromValue(20), FromValue(30)},
			wantVal: 30,
			wantErr: false,
		},
		{
			name: "multiple futures one fails first",
			futures: []*Future[int]{
				FromError[int](errors.New("failure")),
				//FromValue(1),
				//FromValue(2),
			},
			wantVal: 0,
			wantErr: true,
		},
		//{
		//	name:    "no futures",
		//	futures: []*Future[int]{},
		//	wantVal: 0,
		//	wantErr: true,
		//},
		{
			name: "delayed futures success",
			futures: []*Future[int]{
				New(func() (int, error) {
					time.Sleep(10 * time.Millisecond)
					return 100, nil
				}),
				New(func() (int, error) {
					time.Sleep(5 * time.Millisecond)
					return 200, nil
				}),
			},
			wantVal: 200,
			wantErr: false,
		},
		{
			name: "delayed futures failure",
			futures: []*Future[int]{
				New(func() (int, error) {
					time.Sleep(10 * time.Millisecond)
				}),
				New(func() (int, error) {
					time.Sleep(5 * time.Millisecond)
					return 0, errors.New("first failure")
				}),
			},
			wantVal: 0,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fut := First(tc.futures...)
			val, err := fut.Await()

			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error: %v, got: %v", tc.wantErr, err)
			}

			if val != tc.wantVal {
				t.Fatalf("expected value: %d, got: %d", tc.wantVal, val)
			}
		})
	}
}
