#!/bin/bash

export END


function calculate_short() {
	# The goal is to include 5 minutes worth of tests.
	case "$1" in
		ALIDNS) END=98 ;;
		AZURE_DNS) END=55 ;;
		CLOUDFLAREAPI) END=41 ;;
		CNR) END=46 ;;
		GANDI_V5) END=80 ;;
		HEDNS) END=20 ;;
		HOSTINGDE) END=23 ;;
		MYTHICBEASTS) END=97 ;;
		NETLIFY) END=50 ;;
		TRANSIP) END=80 ;;
		VULTR) END=10 ;;

		*) END=999 ;;
	esac
}

function calculate_long() {
	# The goal is to include all tests, but no more than 30 minutes.
	case "$1" in

		HEDNS) END=50 ;;
		HOSTINGDE) END=80 ;;
		VULTR) END=93 ;;

		*) END=999 ;;
	esac
}

function calculate_capped() {
	# Overrides for specific providers.
	case "$1" in
		BIND) # BIND is very fast. We always run all the tests.
			END=999 ;;

		VERCEL) # VERCEL it limited to 100 updates per hour. Never run more than the first few tests.
	       		END=3 ;;

	esac
}

calculate_"$1" "$2"
calculate_capped
echo "$END"

exit 0
