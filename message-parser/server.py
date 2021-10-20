import logging
from concurrent.futures import ThreadPoolExecutor

import grpc

from message_parser_pb2 import ParserResponse
from message_parser_pb2_grpc import MessageParserServicer, add_MessageParserServicer_to_server

import os
import sushy
import json
from sushy import auth
from sushy.resources import base
from sushy.resources.registry import message_registry

# disable InsecureRequestWarning: Unverified HTTPS request is being made to host
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

MSG_PARSER_PORT = 9097

def get_log_level(level):
    level = level.upper()
    if level == "DEBUG":
        return logging.DEBUG
    elif level == "INFO":
        return logging.INFO
    elif level == "WARNING":
        return logging.WARNING
    elif level == "ERROR":
        return logging.ERROR
    elif level == "TRACE":
        return logging.DEBUG
    else:
        logging.warning('Log level %s is not supported. Set level to DEBUG.', level)
        return logging.DEBUG

class MessageParserServicer(MessageParserServicer):

    def __init__(self):
        redfish_username = os.environ.get('REDFISH_USERNAME')
        redfish_password = os.environ.get('REDFISH_PASSWORD')
        redfish_hostaddr = os.environ.get('REDFISH_HOSTADDR')

        basic_auth = auth.BasicAuth(username=redfish_username, password=redfish_password)
        self.sushy_root = sushy.Sushy('https://' + redfish_hostaddr + '/redfish/v1',
                auth=basic_auth, verify=False)
        logging.info('Redfish version: %s', self.sushy_root.redfish_version)
        self.registries = self.sushy_root.lazy_registries
        # preload the registries
        logging.info('Preloading Redfish Registries...')
        self.registries.registries        
        logging.info('Preloading Redfish Registries DONE')
    
    def Parse(self, request, context):
        logging.debug('request message_id: %s', request.message_id)
        logging.debug('request %d message_args', len(request.message_args))
        for a in request.message_args:
            logging.debug('found message arg %s', a)

        m = base.MessageListField('Message')
        m.message_id = request.message_id
        m.message_args = request.message_args
        m.severity = None
        m.resolution = None

        message_registry.parse_message(self.registries, m)
        resp = ParserResponse(message=m.message, severity=m.severity, resolution=m.resolution)
        logging.debug('resp: %s', resp)
        return resp


if __name__ == '__main__':
    l = os.environ.get('LOG_LEVEL', 'DEBUG')
    log_level= get_log_level(l)
    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(levelname)s - %(message)s',
    )
    LOG = logging.getLogger('sushy')
    # Minimize log level for sushy to improve performance
    LOG.setLevel(logging.WARNING)
    LOG.addHandler(logging.StreamHandler())

    server = grpc.server(ThreadPoolExecutor())
    add_MessageParserServicer_to_server(MessageParserServicer(), server)

    msg_parser_port = os.environ.get('MSG_PARSER_PORT')
    port = msg_parser_port if msg_parser_port else MSG_PARSER_PORT
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    logging.info('server ready on port %r', port)
    server.wait_for_termination()

