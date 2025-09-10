package utils

import (
	"github.com/cyinnove/logify"
)

// PrintBanner prints a simple ASCII logo banner to stdout.
func PrintBanner() {
    const banner = `
   __  __                 _   
   \ \/ _\ ___ ___  _   _| |_ 
    \ \ \ / __/ _ \| | | | __|
 /\_/ /\ \ (_| (_) | |_| | |_ 
 \___/\__/\___\___/ \__,_|\__|  @CyInnove
    
Fast, scope-aware, headless crawling framework to extract Dynamic JS files.

`
	logify.Silentf("%s", banner)
}
