  ACTUAL="$1"
  EXPECTED="$2"
  if [[ "$?" -ne 0 ]]; then
      echo "🔴 Unexpected non-zero return code"
      exit 1
  fi
  if [[ "$ACTUAL" != "$EXPECTED" ]]; then
      echo "🔴 Expected '${EXPECTED}' but got '${ACTUAL}'"
      exit 1
  fi

  echo "🟢 Passed"