"""The Python implementation of the gRPC route guide client."""

from __future__ import print_function

import random
import argparse
import logging
import os
import sys

import grpc

sys.path.insert(0, os.path.dirname(
    os.path.realpath(__file__)) + '/../')
sys.path.insert(0, os.path.dirname(
    os.path.realpath(__file__)) + '/../beerlikes')

from google.protobuf.json_format import MessageToJson, Parse
import beerlikes.beer_likes_pb2 as bl_pb2
import beerlikes.beer_likes_pb2_grpc as bl_pb2_grpc

logger = logging.getLogger(__name__)


def get_one_like(stub, like):
    try:
        like = stub.GetLike(like)
    except grpc.RpcError as error:
        logger.warning(error)
        return
    if not like.id:
        logger.warning("Server returned incomplete like")
        return
    logger.debug("Like called %s at %s" % (like.id, like.ref_type))


def get_like(stub):
    get_one_like(stub, bl_pb2.LikeQuery(id="3e8f9d58-4148-4809-9392-63e90fbc8280"))
    get_one_like(stub, bl_pb2.LikeQuery(id="123-abc"))
    get_one_like(stub, bl_pb2.LikeQuery())


def list_likes(stub):
    id="1"
    query = {
        'id': '1',
        'name': 'beer'
        }
    req = bl_pb2.LikesQuery(ref_type=bl_pb2.RefType(**query))
    logger.debug("Looking for likes at %s", query)

    likes = stub.ListLikes(req)

    like_summary = bl_pb2.LikesSummary()
    for item in likes:
        logger.debug("Like called %s %s", type(item), MessageToJson(item))
        ## There is an error with protobuf that cannot compare the classes
        ## https://github.com/protocolbuffers/protobuf/issues/4928. For now we have to
        ## rebuild the Message from JSON
        # like_summary.likes.extend([item])
        like_summary.likes.extend([Parse(MessageToJson(item), bl_pb2.Like())])
        if item.liked:
            like_summary.total += 1
        else:
            like_summary.total -= 1
    logger.debug("LikeSummary %s", MessageToJson(like_summary))


def run(**kwargs):
    # NOTE(gRPC Python Team): .close() is possible on a channel and should be
    # used in circumstances in which the with statement does not fit the needs
    # of the code.
    host = kwargs.get('host', None)
    port = kwargs.get('port', None)
    with grpc.insecure_channel('%s:%s' % (host, port)) as channel:
        stub = bl_pb2_grpc.BeerLikesStub(channel)
        logger.info("-------------- GetLike --------------")
        get_like(stub)
        logger.info("-------------- ListLikes --------------")
        list_likes(stub)


if __name__ == '__main__':
    logging.basicConfig(
        level=logging.DEBUG,
        format=("%(asctime)s %(levelname)s %(name)s[%(process)s] : %(funcName)s"
                " : %(message)s"),
    )
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", help="Hostname or IP address", dest="host",
                        type=str, default='0.0.0.0')
    parser.add_argument("--port", help="Port number", dest="port", type=int,
                        default=10000)
    args = parser.parse_args()
    print(args)
    run(**args.__dict__)
