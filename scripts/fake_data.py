#!/usr/bin/env python3

import math
import time
import sys
import argparse
import random

parser = argparse.ArgumentParser()
parser.add_argument("-n", "--num-columns", type=int, default=1)
parser.add_argument("--csv", action="store_true", default=False)
args = parser.parse_args()

amplitude = 10
shift = 5
period = 10
sleep_time = 0.5

amplitude_noise = 2
period_noise = 2

start = time.time()

separator = "   "
if args.csv:
  separator = ","

while True:
  y = amplitude * math.sin(2 * math.pi * (time.time() - start) / period) + shift
  data = []

  for i in range(args.num_columns):
    data.append(
      str(amplitude * math.sin(2 * math.pi * (time.time() - start) / period) + shift + random.random() * amplitude_noise * math.sin(2 * math.pi * (time.time() - start) / random.random() * period_noise))
    )

  print(separator.join(data))
  sys.stdout.flush()
  time.sleep(sleep_time)
