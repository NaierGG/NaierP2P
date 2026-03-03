import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../shared/models/chat_message.dart';
import '../messages/message_input.dart';
import '../messages/message_list.dart';

class ChannelDetailScreen extends StatefulWidget {
  const ChannelDetailScreen({
    super.key,
    required this.channelId,
  });

  final String channelId;

  @override
  State<ChannelDetailScreen> createState() => _ChannelDetailScreenState();
}

class _ChannelDetailScreenState extends State<ChannelDetailScreen> {
  final Random _random = Random();
  late List<ChatMessage> _messages;
  ChatMessage? _replyTarget;
  ChatMessage? _editingMessage;
  bool _isLoadingOlder = false;
  bool _isRemoteTyping = false;

  @override
  void initState() {
    super.initState();
    _messages = _seedMessages(widget.channelId);
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
            Text('Channel ${widget.channelId}'),
            Text(
              _isRemoteTyping
                  ? 'Someone is typing...'
                  : '$membersOnline members online',
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
        child: Column(
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
                    child: const Text('Ayla is typing...'),
                  ),
                ),
              ),
            Expanded(
              child: MessageListWidget(
                messages: _messages,
                isLoadingOlder: _isLoadingOlder,
                onLoadOlder: _loadOlderMessages,
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

  Future<void> _handleSend(MessageInputResult result) async {
    if (result.editingMessage != null) {
      setState(() {
        _messages = _messages
            .map(
              (message) => message.id == result.editingMessage!.id
                  ? message.copyWith(
                      content: result.text,
                      isEdited: true,
                    )
                  : message,
            )
            .toList();
        _editingMessage = null;
      });
      return;
    }

    final createdAt = DateTime.now();
    final pendingMessage = ChatMessage(
      id: result.clientEventId,
      channelId: widget.channelId,
      senderId: 'me',
      senderName: 'You',
      content: result.text.isEmpty
          ? (result.attachment?.label ?? '')
          : result.text,
      createdAt: createdAt,
      type: _mapAttachmentToMessageType(result.attachment?.kind),
      attachmentLabel: result.attachment?.label,
      replyPreview: result.replyTo?.content,
      isOwn: true,
      status: ChatMessageDeliveryStatus.sending,
    );

    setState(() {
      _messages = [..._messages, pendingMessage];
      _replyTarget = null;
    });

    await Future<void>.delayed(const Duration(milliseconds: 550));
    if (!mounted) {
      return;
    }

    final didFail = _random.nextInt(10) == 0;
    setState(() {
      _messages = _messages
          .map(
            (message) => message.id == pendingMessage.id
                ? message.copyWith(
                    id: 'msg-${createdAt.microsecondsSinceEpoch}',
                    status: didFail
                        ? ChatMessageDeliveryStatus.failed
                        : ChatMessageDeliveryStatus.sent,
                  )
                : message,
          )
          .toList();
    });
  }

  Future<void> _loadOlderMessages() async {
    setState(() => _isLoadingOlder = true);
    await Future<void>.delayed(const Duration(milliseconds: 700));
    if (!mounted) {
      return;
    }

    final oldest = _messages.first.createdAt;
    final olderMessages = List<ChatMessage>.generate(8, (index) {
      final createdAt = oldest.subtract(Duration(minutes: (index + 1) * 14));
      final isOps = index.isEven;
      return ChatMessage(
        id: 'older-${createdAt.microsecondsSinceEpoch}',
        channelId: widget.channelId,
        senderId: isOps ? 'ops' : 'ayla',
        senderName: isOps ? 'Ops' : 'Ayla',
        content: isOps
            ? 'Backfilled secure message ${index + 1}'
            : 'History page ${index + 1} loaded with cursor pagination.',
        createdAt: createdAt,
        type: ChatMessageType.text,
        isOwn: false,
        status: ChatMessageDeliveryStatus.read,
      );
    }).reversed.toList();

    setState(() {
      _messages = [...olderMessages, ..._messages];
      _isLoadingOlder = false;
    });
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
      _messages = _messages.where((item) => item.id != message.id).toList();
      if (_editingMessage?.id == message.id) {
        _editingMessage = null;
      }
      if (_replyTarget?.id == message.id) {
        _replyTarget = null;
      }
    });
  }

  void _toggleReaction(ChatMessage message, String emoji) {
    setState(() {
      _messages = _messages.map((item) {
        if (item.id != message.id) {
          return item;
        }

        final existingIndex =
            item.reactions.indexWhere((reaction) => reaction.emoji == emoji);
        final updated = [...item.reactions];
        if (existingIndex == -1) {
          updated.add(
            ChatMessageReaction(
              emoji: emoji,
              count: 1,
              isMine: true,
            ),
          );
        } else {
          final existing = updated[existingIndex];
          if (existing.isMine && existing.count <= 1) {
            updated.removeAt(existingIndex);
          } else {
            updated[existingIndex] = ChatMessageReaction(
              emoji: existing.emoji,
              count: existing.isMine ? existing.count - 1 : existing.count + 1,
              isMine: !existing.isMine,
            );
          }
        }

        return item.copyWith(reactions: updated);
      }).toList();
    });
  }

  void _handleTypingChanged(bool isTyping) {
    if (!mounted) {
      return;
    }

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
