#!/usr/bin/env python3

import math
import time
import sys

amplitude = 10
shift = 5
period = 10
sleep_time = 0.5

start = time.time()

while True:
  y = amplitude * math.sin(2 * math.pi * (time.time() - start) / period) + shift
  print(y)
  sys.stdout.flush()
  time.sleep(sleep_time)
