#!/usr/bin/env python3

# verify event message contents

import json
import re
import sys
import logging

LOG_DIR = './logs'
FILE_EVENT_FIELDS_TO_VERIFY = './e2e-tests/data/EVENT_FIELDS_TO_VERIFY'

def main():
    logging.basicConfig(filename=LOG_DIR + '/verify-sanity.log', filemode='w', level=logging.DEBUG)
    event_data_file = sys.argv[1]
    consumer_log_file = sys.argv[2]
    event_received = None
    with open(event_data_file, 'r') as f:
        event_expected = json.load(f)['Events'][0]
    if not event_expected:
        print("Error: events data is missing in file {}".format(event_data_file))
        sys.exit(1)
    logging.debug("Expected event: %s", event_expected)

    with open(FILE_EVENT_FIELDS_TO_VERIFY, 'r') as f:
        fields_to_verify = json.load(f)['keys']

    with open(consumer_log_file, 'r') as f:
        for event_received in f:
            match_found = True
            for k, v in event_expected.items():
                if k in fields_to_verify and compare(event_received, k, v) != 0:
                    match_found = False
                    break
            if match_found:
                print("Match found for {}".format(event_data_file))
                sys.exit(0)
    print("Match not found for {}".format(event_data_file))
    sys.exit(1)
    

def compare(received, k, expected):
    pattern = None
    if k == 'MessageArgs':
        pattern = ',{}:\[(.*?)\]'.format(k)
    else:
        pattern = ',{}:(.*?)[,}}]'.format(k)
    m = re.search(pattern, received)
    if not m:
        logging.debug("key %s not found. Pattern used: %s", k, pattern)
        return -1
    actual = m.group(1)
    if k == 'MessageArgs':
        args = actual.split(',')
        if set(args) != set(expected):
            logging.debug("key: %s, expected: %s, actual: %s", k, expected, args)
            return -1
    elif actual != expected:
        logging.debug("key: %s, expected: %s, actual: %s", k, expected, actual)
        return -1
    return 0

if __name__ == '__main__':
    main()