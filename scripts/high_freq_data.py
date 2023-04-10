import time
import math
import sys

HZ = 100
INTERVAL = 1 / HZ

amplitude = 10
shift = 5
period = 20

def f(x: float) -> float:
  return amplitude * math.sin(2 * math.pi * x / period) + shift

start = time.time()

while True:
  t = time.time()
  x = t - start
  y = f(x)
  print(f"{t} {y}")
  time.sleep(INTERVAL)
  sys.stdout.flush()
