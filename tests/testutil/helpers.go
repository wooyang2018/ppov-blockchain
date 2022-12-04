// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package testutil

import (
	"fmt"
	"time"
)

// Sleep print duration and call time.Sleep
func Sleep(d time.Duration) {
	fmt.Printf("Wait for %s\n", d)
	time.Sleep(d)
}
