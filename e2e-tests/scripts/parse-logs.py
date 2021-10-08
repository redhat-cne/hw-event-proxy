#!/usr/bin/env python3

from datetime import datetime
import tempfile
import os
import re
import shutil
import sys

LOG_DIR = '/home/jacding/logs/'
LOG_FILE = '/_latency.log'
REPORT_FILE = '/_report.csv'

LATENCY_RANGES = [ 1, 2, 3, 4, 5, 10, 15, 20, 30, 40, 50, 100, 200, 500, 1000]
# indicate the latency is larger than the maximum range
LATENCY_MAX = 99999

class Report:
    def __init__(self):
        self.latency = dict()
        for i in LATENCY_RANGES:
            self.latency[i] = 0
        self.latency[LATENCY_MAX] = 0
        self.total_msgs = 0
        self.timestamp_first = None
        self.timestamp_last = None
        self.shortest = None
        self.longest = None
        # number of messages within 10ms latency
        self.within10ms = 0
        self.within20ms = 0
        self.within100ms = 0

# parse a RFC3339 format timestamp"2021-08-19T18:26:19Z"
def parsetime(str):
    return datetime.strptime(str,"%Y-%m-%dT%H:%M:%SZ")

# example line:
# time="2021-08-19T18:26:19Z" level=info msg="Latency for hardware event: 2 ms\n"
def parseline(line, report):
    # m = re.search('Latency for hardware event: (\d+) ms', line)
    m = re.search('time="([^"]+)" level=info msg="Latency for hardware event: (\d+) ms', line)
    if m:
        l = int(m.group(2))
        t = m.group(1)
        if report.timestamp_first == None:
            report.timestamp_first = parsetime(t)
        report.timestamp_last = parsetime(t)
        report.total_msgs += 1
        if report.shortest == None:
            report.shortest = l
            report.longest = l
        if l < report.shortest:
            report.shortest = l
        if l > report.longest:
            report.longest = l
        if l <= 10:
            report.within10ms += 1
        if l <= 20:
            report.within20ms += 1
        if l <= 100:
            report.within100ms += 1
        for i in LATENCY_RANGES:
            if l < i:
                report.latency[i] += 1
                return
        # last 
        report.latency[LATENCY_MAX] += 1

def main():
    log_dir = sys.argv[1] if len(sys.argv) > 1 else LOG_DIR
    log_file = log_dir + LOG_FILE
    report_file = log_dir + REPORT_FILE
    report = Report()
    with open(log_file, 'r') as reader:
        line = reader.readline()
        parseline(line, report)
        #the EOF char is an empty string
        while line != '':
            # print(line, end='')
            line = reader.readline()
            parseline(line, report)

    time_elapsed = report.timestamp_last - report.timestamp_first

    with tempfile.NamedTemporaryFile(mode='w', delete=False) as tmp_file:
        tmp_file.write("Total Events\t{}\n".format(report.total_msgs))
        tmp_file.write("Events per Second\t{:.2f}\n".format(report.total_msgs/time_elapsed.total_seconds()))
        tmp_file.write("Shortest Latency\t{}ms\n".format(report.shortest))
        tmp_file.write("Longest Latency\t{}ms\n".format(report.longest))
        tmp_file.write("Percentage within 10ms\t{:.2%}\n".format(report.within10ms/report.total_msgs))
        tmp_file.write("Percentage within 20ms\t{:.2%}\n".format(report.within20ms/report.total_msgs))
        tmp_file.write("Percentage within 100ms\t{:.2%}\n".format(report.within100ms/report.total_msgs))
        tmp_file.write("Time Elapsed\t{}\n".format(time_elapsed))
        tmp_file.write("Time Start\t{}\n".format(report.timestamp_first))
        tmp_file.write("Time End\t{}\n".format(report.timestamp_last))
        tmp_file.write("Latency(<ms)\tNumber of events\t%\n")
        for k in sorted(report.latency):
            if k != LATENCY_MAX:
                 tmp_file.write("{}\t{}\n".format(k, report.latency[k]))
            else:
                 tmp_file.write("{}+\t{}\n".format(LATENCY_RANGES[-1], report.latency[k]))
    #shutil.copystat(report_filename, tmp_file.name)
    os.chmod(tmp_file.name, 0o644)
    shutil.move(tmp_file.name, report_file)
    print("output to: {}\n".format(report_file))

if __name__ == '__main__':
    main()