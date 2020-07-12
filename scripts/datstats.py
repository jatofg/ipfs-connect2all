import sys
import csv
import math

if len(sys.argv) < 2:
	print("Usage: python3 datstats.py STATSFILE.dat")
	sys.exit()

colVals = []
colSums = []
numCols = 0
initialized = False

def initialize(n):
	global initialized, numCols, colVals, colSums
	initialized = True
	numCols = n
	for i in range(n):
		colVals.append([])
		colSums.append(0.)

crawlP = []
crawlReachableP = []
crawlSum = 0.
crawlReachableSum = 0.
with open(sys.argv[1], newline='') as csvfile:
	creader = csv.reader(csvfile, delimiter='\t')
	for row in creader:
		if not initialized:
			initialize(len(row))
		for i in range(numCols):
			colVals[i].append(float(row[i]))
			colSums[i] += float(row[i])
for i in range(numCols):
	colVals[i].sort()
	print("Col {} mean: {}".format(i+1, colSums[i]/len(colVals[i])))
	print("Col {} median: {}".format(i+1, colVals[i][math.floor(len(colVals[i])/2)]))
	print("Col {} min/max: {} / {}".format(i+1, colVals[i][0], colVals[i][-1]))
	print()
