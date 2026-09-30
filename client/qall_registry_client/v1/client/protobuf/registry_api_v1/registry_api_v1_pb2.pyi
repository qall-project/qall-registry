from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import wrappers_pb2 as _wrappers_pb2
from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import (
    ClassVar as _ClassVar,
    Iterable as _Iterable,
    Mapping as _Mapping,
    Optional as _Optional,
    Union as _Union,
)

DESCRIPTOR: _descriptor.FileDescriptor

class Block(_message.Message):
    __slots__ = ("hash", "data")
    HASH_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    hash: str
    data: bytes
    def __init__(
        self, hash: _Optional[str] = ..., data: _Optional[bytes] = ...
    ) -> None: ...

class Tag(_message.Message):
    __slots__ = ("name", "version", "root_hash")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ROOT_HASH_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    root_hash: str
    def __init__(
        self,
        name: _Optional[str] = ...,
        version: _Optional[str] = ...,
        root_hash: _Optional[str] = ...,
    ) -> None: ...

class RegistryInfo(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    def __init__(
        self, name: _Optional[str] = ..., version: _Optional[str] = ...
    ) -> None: ...

class CheckRequest(_message.Message):
    __slots__ = ("hashes",)
    HASHES_FIELD_NUMBER: _ClassVar[int]
    hashes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, hashes: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckResponse(_message.Message):
    __slots__ = ("missing_hashes", "total_count")
    MISSING_HASHES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    missing_hashes: _containers.RepeatedScalarFieldContainer[str]
    total_count: int
    def __init__(
        self,
        missing_hashes: _Optional[_Iterable[str]] = ...,
        total_count: _Optional[int] = ...,
    ) -> None: ...

class PushRequest(_message.Message):
    __slots__ = ("blocks",)
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    blocks: _containers.RepeatedCompositeFieldContainer[Block]
    def __init__(
        self, blocks: _Optional[_Iterable[_Union[Block, _Mapping]]] = ...
    ) -> None: ...

class PushResponse(_message.Message):
    __slots__ = ("written_blocks_count",)
    WRITTEN_BLOCKS_COUNT_FIELD_NUMBER: _ClassVar[int]
    written_blocks_count: int
    def __init__(self, written_blocks_count: _Optional[int] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("hashes",)
    HASHES_FIELD_NUMBER: _ClassVar[int]
    hashes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, hashes: _Optional[_Iterable[str]] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("blocks", "total_count")
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    blocks: _containers.RepeatedCompositeFieldContainer[Block]
    total_count: int
    def __init__(
        self,
        blocks: _Optional[_Iterable[_Union[Block, _Mapping]]] = ...,
        total_count: _Optional[int] = ...,
    ) -> None: ...

class TagRequest(_message.Message):
    __slots__ = ("tag",)
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: Tag
    def __init__(self, tag: _Optional[_Union[Tag, _Mapping]] = ...) -> None: ...

class TagResponse(_message.Message):
    __slots__ = ("uri",)
    URI_FIELD_NUMBER: _ClassVar[int]
    uri: str
    def __init__(self, uri: _Optional[str] = ...) -> None: ...

class ResolveRequest(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    def __init__(
        self, name: _Optional[str] = ..., version: _Optional[str] = ...
    ) -> None: ...

class ResolveResponse(_message.Message):
    __slots__ = ("root_hash",)
    ROOT_HASH_FIELD_NUMBER: _ClassVar[int]
    root_hash: str
    def __init__(self, root_hash: _Optional[str] = ...) -> None: ...

class ListTagsRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ListTagsResponse(_message.Message):
    __slots__ = ("tags", "total_count")
    TAGS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    tags: _containers.RepeatedCompositeFieldContainer[Tag]
    total_count: int
    def __init__(
        self,
        tags: _Optional[_Iterable[_Union[Tag, _Mapping]]] = ...,
        total_count: _Optional[int] = ...,
    ) -> None: ...

class UntagRequest(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    def __init__(
        self, name: _Optional[str] = ..., version: _Optional[str] = ...
    ) -> None: ...

class UntagResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: bool = ...) -> None: ...
