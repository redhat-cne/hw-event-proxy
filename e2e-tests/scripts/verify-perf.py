#!/usr/bin/env python3

# parse logs for multiple consumers.

from datetime import datetime
import tempfile
import os
import re
import shutil
import sys
import glob

LOG_DIR = './logs'
CONSUMER_POD_NAME = 'consumer'
REPORT_FILE = '_report.csv'
VERBOSE = False

LATENCY_RANGES = [ 1, 2, 3, 4, 5, 10, 15, 20, 30, 40, 50, 100, 200, 500, 1000]
# indicate the latency is larger than the maximum range
LATENCY_MAX = 99999

# pattern to search in the consumer logs:
# example line: time="2021-08-19T18:26:19Z" level=info msg="Latency for hardware event: 2 ms\n"
LATENCY_PATTERN = 'time="([^"]+)" level=info msg="Latency for the event: (\d+) ms'

def log_debug(log):
    if VERBOSE:
        print(log)

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
        self.consumers = 0

    def generate_report(self, report_file):
        time_elapsed = self.timestamp_last - self.timestamp_first

        with tempfile.NamedTemporaryFile(mode='w', delete=False) as tmp_file:
            tmp_file.write("Number of Consumers\t{}\n".format(self.consumers))
            tmp_file.write("Events per Consumer\t{:.0f}\n".format(self.total_msgs/self.consumers))

            # for multiple consumers report Events/Second per user
            tmp_file.write("Events per Second\t{:.2f}\n".format(self.total_msgs/self.consumers/time_elapsed.total_seconds()))
            tmp_file.write("Shortest Latency\t{}ms\n".format(self.shortest))
            tmp_file.write("Longest Latency\t{}ms\n".format(self.longest))
            tmp_file.write("Percentage within 10ms\t{:.2%}\n".format(self.within10ms/self.total_msgs))
            tmp_file.write("Percentage within 20ms\t{:.2%}\n".format(self.within20ms/self.total_msgs))
            tmp_file.write("Percentage within 100ms\t{:.2%}\n".format(self.within100ms/self.total_msgs))
            tmp_file.write("Time Elapsed\t{}\n".format(time_elapsed))
            tmp_file.write("Time Start\t{}\n".format(self.timestamp_first))
            tmp_file.write("Time End\t{}\n".format(self.timestamp_last))
            tmp_file.write("Latency(<ms)\tNumber of events\t%\n")
            for k in sorted(self.latency):
                if k != LATENCY_MAX:
                    tmp_file.write("{}\t{}\n".format(k, self.latency[k]))
                else:
                    tmp_file.write("{}+\t{}\n".format(LATENCY_RANGES[-1], self.latency[k]))
        os.chmod(tmp_file.name, 0o644)
        shutil.move(tmp_file.name, report_file)
        log_debug("output to: {}".format(report_file))

    def parseline(self, line):
        m = re.search(LATENCY_PATTERN, line)
        if m:
            l = int(m.group(2))
            t = m.group(1)
            if self.timestamp_first == None:
                self.timestamp_first = parsetime(t)
            self.timestamp_last = parsetime(t)
            self.total_msgs += 1
            if self.shortest == None:
                self.shortest = l
                self.longest = l
            if l < self.shortest:
                self.shortest = l
            if l > self.longest:
                self.longest = l
            if l <= 10:
                self.within10ms += 1
            if l <= 20:
                self.within20ms += 1
            if l <= 100:
                self.within100ms += 1
            for i in LATENCY_RANGES:
                if l < i:
                    self.latency[i] += 1
                    return
            # last
            self.latency[LATENCY_MAX] += 1

    def parse_file(self, log_file):
        with open(log_file, 'r') as reader:
            line = reader.readline()
            self.parseline(line)
            #the EOF char is an empty string
            while line != '':
                # print(line, end='')
                line = reader.readline()
                self.parseline(line)

# parse a RFC3339 format timestamp"2021-08-19T18:26:19Z"
def parsetime(str):
    return datetime.strptime(str,"%Y-%m-%dT%H:%M:%SZ")


def report_each_file(log_dir, log_files):
    report = Report()
    for log_file in log_files:
        file_tag = None
        m = re.search('(-\w+)\.log$', log_file)
        if m:
            file_tag = m.group(1)
        else:
            print("Error: file pattern does not match _latency_consumer-65ff4ccc65-4n9n8.log")
            return
        report_file = log_dir + '/_report' + file_tag + '.csv'
        report.parse_file(log_file)
        if report.timestamp_first is None:
            print("Error: log {} does not contain any latency info".format(log_file))
            sys.exit(1)
        report.generate_report(report_file)


def report_all(log_dir, log_files):
    report = Report()
    report_file = log_dir + '/' + REPORT_FILE
    for log_file in log_files:
        report.consumers += 1
        report.parse_file(log_file)
    if report.timestamp_first is None:
        print("Error: logs does not contain any latency info")
        sys.exit(1)
    report.generate_report(report_file)

def main():
    log_dir = sys.argv[1] if len(sys.argv) > 1 else LOG_DIR
    log_files = glob.glob(log_dir + '/' + CONSUMER_POD_NAME + '*.log')

    # report_each_file(log_dir, log_files)
    report_all(log_dir, log_files)


if __name__ == '__main__':
    main()