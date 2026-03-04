import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_client.dart';
import '../../shared/models/chat_message.dart';
import '../messages/message_input.dart';
import '../messages/message_list.dart';

class ChannelDetailScreen extends ConsumerStatefulWidget {
  const ChannelDetailScreen({
    super.key,
    required this.channelId,
  });

  final String channelId;

  @override
  ConsumerState<ChannelDetailScreen> createState() => _ChannelDetailScreenState();
}

class _ChannelDetailScreenState extends ConsumerState<ChannelDetailScreen> {
  final List<ChatMessage> _messages = <ChatMessage>[];
  final Map<String, _ChannelMemberSummary> _members = <String, _ChannelMemberSummary>{};
  StreamSubscription<Map<String, dynamic>>? _wsSubscription;
  ChatMessage? _replyTarget;
  ChatMessage? _editingMessage;
  String? _cursor;
  String _channelTitle = '';
  String _channelSubtitle = '';
  bool _hasMore = true;
  bool _isLoading = true;
  bool _isLoadingOlder = false;
  bool _isRemoteTyping = false;
  bool _usingDemoData = false;

  @override
  void initState() {
    super.initState();
    unawaited(_bootstrap());
  }

  @override
  void dispose() {
    _wsSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final membersOnline = _messages.map((message) => message.senderId).toSet().length;

    return Scaffold(
      appBar: AppBar(
        titleSpacing: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(_channelTitle.isNotEmpty ? _channelTitle : 'Channel ${widget.channelId}'),
            Text(
              _isRemoteTyping
                  ? 'Someone is typing...'
                  : _usingDemoData
                      ? 'Demo mode'
                      : (_channelSubtitle.isNotEmpty ? _channelSubtitle : '$membersOnline members online'),
              style: Theme.of(context).textTheme.labelMedium,
            ),
          ],
        ),
        actions: [
          IconButton(
            onPressed: () {},
            icon: const Icon(Icons.call_outlined),
          ),
          IconButton(
            onPressed: () {},
            icon: const Icon(Icons.more_horiz_rounded),
          ),
        ],
      ),
      body: DecoratedBox(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              Theme.of(context).colorScheme.surface,
              Theme.of(context).scaffoldBackgroundColor,
            ],
          ),
        ),
        child: _isLoading
            ? const Center(child: CircularProgressIndicator())
            : Column(
                children: [
                  if (_isRemoteTyping)
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                      child: Align(
                        alignment: Alignment.centerLeft,
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 12,
                            vertical: 8,
                          ),
                          decoration: BoxDecoration(
                            color: Theme.of(context).colorScheme.surfaceContainerHigh,
                            borderRadius: BorderRadius.circular(999),
                          ),
                          child: const Text('Someone is typing...'),
                        ),
                      ),
                    ),
                  Expanded(
                    child: MessageListWidget(
                      messages: _messages,
                      isLoadingOlder: _isLoadingOlder,
                      onLoadOlder: _hasMore ? _loadOlderMessages : null,
                      onReply: (message) => setState(() {
                        _editingMessage = null;
                        _replyTarget = message;
                      }),
                      onCopy: _copyMessage,
                      onEdit: (message) => setState(() {
                        _replyTarget = null;
                        _editingMessage = message;
                      }),
                      onDelete: _deleteMessage,
                      onToggleReaction: _toggleReaction,
                    ),
                  ),
                  MessageInputWidget(
                    replyTo: _replyTarget,
                    editingMessage: _editingMessage,
                    onCancelReply: () => setState(() => _replyTarget = null),
                    onCancelEdit: () => setState(() => _editingMessage = null),
                    onTypingChanged: _handleTypingChanged,
                    onSend: _handleSend,
                  ),
                ],
              ),
      ),
    );
  }

  Future<void> _bootstrap() async {
    final wsClient = ref.read(websocketClientProvider);
    wsClient.connect();
    wsClient.send({
      'type': 'CHANNEL_JOIN',
      'payload': {'channelId': widget.channelId},
    });

    _wsSubscription = wsClient.events.listen(_handleWsEvent);
    await _loadInitialMessages();
  }

  Future<void> _loadInitialMessages() async {
    try {
      final dio = ref.read(dioProvider);
      final channelFuture = dio.get<Map<String, dynamic>>('/channels/${widget.channelId}');
      final membersFuture = dio.get<Map<String, dynamic>>('/channels/${widget.channelId}/members');
      final response = await dio.get<Map<String, dynamic>>(
        '/channels/${widget.channelId}/messages',
        queryParameters: {'limit': 40},
      );
      final channelResponse = await channelFuture;
      final membersResponse = await membersFuture;
      final body = response.data ?? <String, dynamic>{};
      final channelBody = channelResponse.data ?? <String, dynamic>{};
      final memberBody = membersResponse.data ?? <String, dynamic>{};
      final members = (memberBody['members'] as List<dynamic>? ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(_ChannelMemberSummary.fromJson)
          .toList();
      final messages = (body['messages'] as List<dynamic>? ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(_messageFromJson)
          .toList()
        ..sort((left, right) => left.createdAt.compareTo(right.createdAt));

      if (!mounted) {
        return;
      }

      setState(() {
        _messages
          ..clear()
          ..addAll(messages);
        _members
          ..clear()
          ..addEntries(
            members.map((member) => MapEntry(member.userId, member)),
          );
        _channelTitle = channelBody['name']?.toString().isNotEmpty == true
            ? channelBody['name'].toString()
            : 'Channel ${widget.channelId}';
        _channelSubtitle = '${members.length} members';
        _cursor = body['next_cursor']?.toString();
        _hasMore = body['has_more'] == true;
        _isLoading = false;
        _usingDemoData = false;
      });
      _emitReadAck();
      return;
    } catch (_) {
      if (!mounted) {
        return;
      }

      setState(() {
        _messages
          ..clear()
          ..addAll(_seedMessages(widget.channelId));
        _members.clear();
        _channelTitle = 'Channel ${widget.channelId}';
        _channelSubtitle = 'Demo mode';
        _cursor = null;
        _hasMore = false;
        _isLoading = false;
        _usingDemoData = true;
      });
    }
  }

  Future<void> _handleSend(MessageInputResult result) async {
    if (result.editingMessage != null) {
      final nextContent = result.text;
      setState(() {
        final index = _messages.indexWhere(
          (message) => message.id == result.editingMessage!.id,
        );
        if (index != -1) {
          _messages[index] = _messages[index].copyWith(
            content: nextContent,
            isEdited: true,
          );
        }
        _editingMessage = null;
      });

      if (!_usingDemoData) {
        ref.read(websocketClientProvider).send({
          'type': 'MESSAGE_EDIT',
          'payload': {
            'messageId': result.editingMessage!.id,
            'content': nextContent,
            'iv': '',
          },
        });
      }
      return;
    }

    final createdAt = DateTime.now();
    final pendingMessage = ChatMessage(
      id: result.clientEventId,
      channelId: widget.channelId,
      senderId: ref.read(authSessionProvider).userId ?? 'me',
      senderName: 'You',
      content: result.text.isEmpty
          ? (result.attachment?.label ?? '')
          : result.text,
      createdAt: createdAt,
      type: _mapAttachmentToMessageType(result.attachment?.kind),
      attachmentLabel: result.attachment?.label,
      replyPreview: result.replyTo?.content,
      clientEventId: result.clientEventId,
      isOwn: true,
      status: ChatMessageDeliveryStatus.sending,
    );

    setState(() {
      _messages.add(pendingMessage);
      _replyTarget = null;
    });

    if (_usingDemoData) {
      await Future<void>.delayed(const Duration(milliseconds: 550));
      if (!mounted) {
        return;
      }
      setState(() {
        _replaceMessage(
          pendingMessage.id,
          pendingMessage.copyWith(
            id: 'msg-${createdAt.microsecondsSinceEpoch}',
            status: ChatMessageDeliveryStatus.sent,
          ),
        );
      });
      return;
    }

    ref.read(websocketClientProvider).send({
      'type': 'MESSAGE_SEND',
      'payload': {
        'channelId': widget.channelId,
        'content': pendingMessage.content,
        'iv': '',
        'clientEventId': result.clientEventId,
      },
    });
  }

  Future<void> _loadOlderMessages() async {
    if (_isLoadingOlder || !_hasMore || _cursor == null || _usingDemoData) {
      return;
    }

    setState(() => _isLoadingOlder = true);
    try {
      final dio = ref.read(dioProvider);
      final response = await dio.get<Map<String, dynamic>>(
        '/channels/${widget.channelId}/messages',
        queryParameters: {
          'cursor': _cursor,
          'limit': 40,
        },
      );
      final body = response.data ?? <String, dynamic>{};
      final olderMessages = (body['messages'] as List<dynamic>? ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(_messageFromJson)
          .toList()
        ..sort((left, right) => left.createdAt.compareTo(right.createdAt));

      if (!mounted) {
        return;
      }

      setState(() {
        _messages.insertAll(0, olderMessages);
        _cursor = body['next_cursor']?.toString();
        _hasMore = body['has_more'] == true;
        _isLoadingOlder = false;
      });
    } catch (_) {
      if (!mounted) {
        return;
      }
      setState(() => _isLoadingOlder = false);
    }
  }

  Future<void> _copyMessage(ChatMessage message) async {
    await Clipboard.setData(ClipboardData(text: message.content));
    if (!mounted) {
      return;
    }

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Message copied')),
    );
  }

  void _deleteMessage(ChatMessage message) {
    setState(() {
      _replaceMessage(
        message.id,
        message.copyWith(
          content: '',
          isEdited: false,
          status: message.status,
        ),
      );
      if (_editingMessage?.id == message.id) {
        _editingMessage = null;
      }
      if (_replyTarget?.id == message.id) {
        _replyTarget = null;
      }
    });

    if (!_usingDemoData) {
      ref.read(websocketClientProvider).send({
        'type': 'MESSAGE_DELETE',
        'payload': {'messageId': message.id},
      });
    }
  }

  void _toggleReaction(ChatMessage message, String emoji) {
    final hasMine = message.reactions.any((reaction) => reaction.emoji == emoji && reaction.isMine);
    if (_usingDemoData) {
      _applyReactionLocally(message.id, emoji, !hasMine);
      return;
    }

    ref.read(websocketClientProvider).send({
      'type': hasMine ? 'REACTION_REMOVE' : 'REACTION_ADD',
      'payload': {
        'messageId': message.id,
        'emoji': emoji,
      },
    });
  }

  void _handleTypingChanged(bool isTyping) {
    if (_usingDemoData) {
      if (isTyping) {
        setState(() => _isRemoteTyping = true);
        unawaited(
          Future<void>.delayed(const Duration(seconds: 3), () {
            if (mounted) {
              setState(() => _isRemoteTyping = false);
            }
          }),
        );
      }
      return;
    }

    ref.read(websocketClientProvider).send({
      'type': isTyping ? 'TYPING_START' : 'TYPING_STOP',
      'payload': {'channelId': widget.channelId},
    });
  }

  void _handleWsEvent(Map<String, dynamic> rawEvent) {
    final type = rawEvent['type']?.toString() ?? '';
    final payload = rawEvent['payload'];
    if (payload is! Map<String, dynamic>) {
      return;
    }

    switch (type) {
      case 'MESSAGE_NEW':
      case 'MESSAGE_UPDATED':
        final message = _messageFromJson(payload);
        if (message.channelId != widget.channelId || !mounted) {
          return;
        }
        setState(() {
          _ackDelivery(message.id);
          final existingIndex = _messages.indexWhere(
            (entry) =>
                entry.id == message.id ||
                (message.clientEventId != null &&
                    entry.clientEventId == message.clientEventId),
          );
          if (existingIndex == -1) {
            _messages.add(message);
          } else {
            _messages[existingIndex] = message.copyWith(
              status: message.isOwn
                  ? ChatMessageDeliveryStatus.sent
                  : _messages[existingIndex].status,
            );
          }
          _messages.sort((left, right) => left.createdAt.compareTo(right.createdAt));
        });
        _emitReadAck();
        break;
      case 'MESSAGE_DELETED':
        final messageId = payload['messageId']?.toString() ?? payload['id']?.toString();
        if (messageId == null || !mounted) {
          return;
        }
        setState(() {
          final index = _messages.indexWhere((entry) => entry.id == messageId);
          if (index != -1) {
            _messages[index] = _messages[index].copyWith(content: '');
          }
        });
        break;
      case 'REACTION':
        final messageId = payload['message_id']?.toString() ?? payload['messageId']?.toString();
        final emoji = payload['emoji']?.toString() ?? '';
        final action = payload['action']?.toString() == 'remove' ? false : true;
        if (messageId == null || emoji.isEmpty || !mounted) {
          return;
        }
        setState(() => _applyReactionLocally(messageId, emoji, action));
        break;
      case 'TYPING':
        final channelId = payload['channelId']?.toString() ?? payload['channel_id']?.toString();
        if (channelId != widget.channelId || !mounted) {
          return;
        }
        setState(() => _isRemoteTyping = payload['isTyping'] == true);
        break;
      default:
        break;
    }
  }

  void _emitReadAck() {
    if (_usingDemoData || _messages.isEmpty) {
      return;
    }

    final lastSequence = _messages
        .where((message) => message.sequence != null)
        .map((message) => message.sequence!)
        .fold<int>(0, (max, value) => value > max ? value : max);
    if (lastSequence == 0) {
      return;
    }

    ref.read(websocketClientProvider).send({
      'type': 'READ_ACK',
      'payload': {
        'channelId': widget.channelId,
        'lastReadSequence': lastSequence,
      },
    });
  }

  void _replaceMessage(String targetId, ChatMessage next) {
    final index = _messages.indexWhere((message) => message.id == targetId);
    if (index == -1) {
      _messages.add(next);
      return;
    }
    _messages[index] = next;
  }

  void _applyReactionLocally(String messageId, String emoji, bool add) {
    final index = _messages.indexWhere((message) => message.id == messageId);
    if (index == -1) {
      return;
    }

    final message = _messages[index];
    final reactions = [...message.reactions];
    final existingIndex = reactions.indexWhere((reaction) => reaction.emoji == emoji);
    if (add) {
      if (existingIndex == -1) {
        reactions.add(ChatMessageReaction(emoji: emoji, count: 1, isMine: true));
      } else {
        final existing = reactions[existingIndex];
        reactions[existingIndex] = ChatMessageReaction(
          emoji: existing.emoji,
          count: existing.count + (existing.isMine ? 0 : 1),
          isMine: true,
        );
      }
    } else if (existingIndex != -1) {
      final existing = reactions[existingIndex];
      if (existing.count <= 1) {
        reactions.removeAt(existingIndex);
      } else {
        reactions[existingIndex] = ChatMessageReaction(
          emoji: existing.emoji,
          count: existing.count - 1,
          isMine: false,
        );
      }
    }

    _messages[index] = message.copyWith(reactions: reactions);
  }

  ChatMessage _messageFromJson(Map<String, dynamic> json) {
    final session = ref.read(authSessionProvider);
    final senderId = json['sender_id']?.toString() ?? '';
    final isOwn = senderId == session.userId;
    final typeValue = json['type']?.toString() ?? 'text';
    final replyTo = json['reply_to_id']?.toString();

    return ChatMessage(
      id: json['id']?.toString() ?? '',
      channelId: json['channel_id']?.toString() ?? widget.channelId,
      senderId: senderId,
      senderName: isOwn ? 'You' : _displayNameForSender(senderId),
      content: json['is_deleted'] == true ? '' : (json['content']?.toString() ?? ''),
      createdAt: DateTime.tryParse(json['created_at']?.toString() ?? '') ?? DateTime.now(),
      type: switch (typeValue) {
        'image' => ChatMessageType.image,
        'file' => ChatMessageType.file,
        _ => ChatMessageType.text,
      },
      replyPreview: replyTo?.isNotEmpty == true ? 'Reply to $replyTo' : null,
      attachmentLabel: typeValue == 'file' ? json['content']?.toString() : null,
      clientEventId: json['client_event_id']?.toString(),
      serverEventId: json['server_event_id']?.toString(),
      sequence: (json['sequence'] as num?)?.toInt(),
      isOwn: isOwn,
      isEdited: json['is_edited'] == true,
      status: isOwn
          ? ChatMessageDeliveryStatus.sent
          : ChatMessageDeliveryStatus.read,
      reactions: (json['reactions'] as List<dynamic>? ?? const [])
          .whereType<Map<String, dynamic>>()
          .map(
            (reaction) => ChatMessageReaction(
              emoji: reaction['emoji']?.toString() ?? '',
              count: 1,
              isMine: reaction['user_id']?.toString() == session.userId,
            ),
          )
          .toList(),
    );
  }

  String _displayNameForSender(String senderId) {
    final member = _members[senderId];
    if (member != null && member.displayName.isNotEmpty) {
      return member.displayName;
    }
    if (member != null && member.username.isNotEmpty) {
      return member.username;
    }
    if (senderId.isEmpty) {
      return 'Unknown';
    }
    return 'User ${senderId.substring(0, senderId.length >= 6 ? 6 : senderId.length)}';
  }

  void _ackDelivery(String messageId) {
    if (_usingDemoData) {
      return;
    }

    ref.read(websocketClientProvider).send({
      'type': 'DELIVERY_ACK',
      'payload': {'messageId': messageId},
    });
  }

  ChatMessageType _mapAttachmentToMessageType(MessageAttachmentKind? kind) {
    switch (kind) {
      case MessageAttachmentKind.camera:
      case MessageAttachmentKind.gallery:
        return ChatMessageType.image;
      case MessageAttachmentKind.file:
        return ChatMessageType.file;
      case null:
        return ChatMessageType.text;
    }
  }

  List<ChatMessage> _seedMessages(String channelId) {
    final now = DateTime.now();
    return [
      ChatMessage(
        id: '1',
        channelId: channelId,
        senderId: 'ops',
        senderName: 'Ops',
        content: 'Federated relay is healthy. Media proxies are green.',
        createdAt: now.subtract(const Duration(days: 1, hours: 5)),
        type: ChatMessageType.text,
        reactions: const [
          ChatMessageReaction(emoji: ':ok:', count: 3),
        ],
      ),
      ChatMessage(
        id: '2',
        channelId: channelId,
        senderId: 'ayla',
        senderName: 'Ayla',
        content: 'Uploading the design board for mobile message states.',
        createdAt: now.subtract(const Duration(hours: 2, minutes: 48)),
        type: ChatMessageType.file,
        attachmentLabel: 'mobile-message-states.pdf',
      ),
      ChatMessage(
        id: '3',
        channelId: channelId,
        senderId: 'ayla',
        senderName: 'Ayla',
        content: 'Typing indicator and reply composer are now wired in.',
        createdAt: now.subtract(const Duration(hours: 2, minutes: 42)),
        type: ChatMessageType.text,
      ),
      ChatMessage(
        id: '4',
        channelId: channelId,
        senderId: 'me',
        senderName: 'You',
        content: 'Looks good. Ship the optimistic sending state next.',
        createdAt: now.subtract(const Duration(minutes: 36)),
        type: ChatMessageType.text,
        isOwn: true,
        status: ChatMessageDeliveryStatus.read,
      ),
      ChatMessage(
        id: '5',
        channelId: channelId,
        senderId: 'ayla',
        senderName: 'Ayla',
        content: 'Previewing image attachments on-device before encryption.',
        createdAt: now.subtract(const Duration(minutes: 20)),
        type: ChatMessageType.image,
        replyPreview: 'Looks good. Ship the optimistic sending state next.',
        reactions: const [
          ChatMessageReaction(emoji: ':fire:', count: 2, isMine: true),
          ChatMessageReaction(emoji: ':clap:', count: 1),
        ],
      ),
    ];
  }
}

class _ChannelMemberSummary {
  const _ChannelMemberSummary({
    required this.userId,
    required this.username,
    required this.displayName,
  });

  final String userId;
  final String username;
  final String displayName;

  factory _ChannelMemberSummary.fromJson(Map<String, dynamic> json) {
    return _ChannelMemberSummary(
      userId: json['user_id']?.toString() ?? '',
      username: json['username']?.toString() ?? '',
      displayName: json['display_name']?.toString() ?? '',
    );
  }
}
