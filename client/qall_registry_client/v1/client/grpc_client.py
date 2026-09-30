# Copyright 2026 Scaleway
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from __future__ import annotations

import grpc
from typing import Optional, List

import qall_registry_client.v1.client.protobuf.registry_api_v1.registry_api_v1_pb2 as pb2
import qall_registry_client.v1.client.protobuf.registry_api_v1.registry_api_v1_pb2_grpc as pb2_grpc

from qall_registry_client.v1.client.base import RegistryClient
from qall_registry_client.v1.objects import Block, Tag


class GrpcRegistryClient(RegistryClient):
    def __init__(
        self,
        url: str,
        namespace: Optional[str] = None,
        token: Optional[str] = None,
        channel: Optional[grpc.Channel] = None,
    ):
        self.__url = url
        self.__channel = channel or grpc.insecure_channel(self.__url)
        self.__stub = pb2_grpc.ApiStub(self.__channel)
        self.__token = token
        self.__namespace = namespace

    def check(self, hashes: List[str]) -> List[str]:
        try:
            hashes = hashes if isinstance(hashes, list) else [hashes]
            request = pb2.CheckRequest(hashes=hashes)
            response = self.__stub.Check(request)
            return list(response.missing_hashes)
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to check hashes: {e.details()}") from e

    def get(self, hashes: List[str]) -> List[Block]:
        try:
            hashes = hashes if isinstance(hashes, list) else [hashes]
            request = pb2.GetRequest(hashes=hashes)
            response = self.__stub.Get(request)
            return list(response.blocks)
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to get blocks: {e.details()}") from e

    def push(self, blocks: List[Block]) -> int:
        if not blocks:
            return 0
        try:
            blocks = blocks if isinstance(blocks, list) else [blocks]
            pb_blocks = [pb2.Block(hash=b.hash, data=b.data) for b in blocks]
            request = pb2.PushRequest(blocks=pb_blocks)
            response = self.__stub.Push(request)
            return response.written_blocks_count
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to push block: {e.details()}") from e

    def tag(self, tag: Tag) -> str:
        try:
            proto_tag = pb2.Tag(
                name=tag.name, version=tag.version, root_hash=tag.root_hash
            )
            request = pb2.TagRequest(tag=proto_tag)
            response = self.__stub.Tag(request)
            return response.uri
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to tag: {e.details()}") from e

    def resolve(self, name: str, version: str) -> str:
        try:
            request = pb2.ResolveRequest(name=name, version=version)
            response = self.__stub.Resolve(request)
            return response.root_hash
        except grpc.RpcError as e:
            raise RuntimeError(
                f"Fail to resolve tag {name}:{version}: {e.details()}"
            ) from e

    def list_tags(self, name: str) -> List[Tag]:
        try:
            request = pb2.ListTagsRequest(name=name)
            response = self.__stub.ListTags(request)
            return [
                Tag(name=t.name, version=t.version, root_hash=t.root_hash)
                for t in response.tags
            ]
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to list tags for {name}: {e.details()}") from e

    def untag(self, name: str, version: str) -> bool:
        try:
            request = pb2.UntagRequest(name=name, version=version)
            response = self.__stub.Untag(request)
            return response.deleted
        except grpc.RpcError as e:
            raise RuntimeError(f"Fail to untag {name}:{version}: {e.details()}") from e

    def close(self) -> None:
        if self.__channel:
            self.__channel.close()

    def __enter__(self) -> GrpcRegistryClient:
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()
