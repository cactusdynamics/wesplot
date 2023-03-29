import math
import sys

n = int(sys.argv[2])

for i in range(1, n):
  print("{} {}".format(i, math.log(i)))
