#!/bin/bash

export END

function calculate_long() {
	# The goal is to include all tests, but no more than 30 minutes.
	case "$1" in

		HEDNS) END=10 ;;

		# VERCEL it limited to 100 updates per hour. Never run more than the first few tests.
		VERCEL) END=3 ;;

		*) END=999 ;;
	esac
}


function calculate_short() {
	# The goal is to include 5 minutes worth of tests.
	case "$1" in

		HEDNS) END=10 ;;

		# VERCEL it limited to 100 updates per hour. Never run more than the first few tests.
		VERCEL) END=3 ;;

		*) END=30 ;;
	esac
}

function calculate_capped() {
	# Overrides for specific providers.
	case "$1" in
		BIND)
			# BIND is very fast. We always run all the tests.
			END=999
			;;
	esac
}

calculate_"$1" "$2"
calculate_capped
echo "$END"
