import 'package:flutter/material.dart';

import '../../shared/models/chat_message.dart';
import 'message_bubble.dart';

class MessageListWidget extends StatefulWidget {
  const MessageListWidget({
    super.key,
    required this.messages,
    this.onLoadOlder,
    this.onReply,
    this.onCopy,
    this.onEdit,
    this.onDelete,
    this.onToggleReaction,
    this.isLoadingOlder = false,
  });

  final List<ChatMessage> messages;
  final Future<void> Function()? onLoadOlder;
  final ValueChanged<ChatMessage>? onReply;
  final ValueChanged<ChatMessage>? onCopy;
  final ValueChanged<ChatMessage>? onEdit;
  final ValueChanged<ChatMessage>? onDelete;
  final void Function(ChatMessage, String)? onToggleReaction;
  final bool isLoadingOlder;

  @override
  State<MessageListWidget> createState() => _MessageListWidgetState();
}

class _MessageListWidgetState extends State<MessageListWidget> {
  final ScrollController _scrollController = ScrollController();
  bool _requestedOlder = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_handleScroll);
  }

  @override
  void dispose() {
    _scrollController
      ..removeListener(_handleScroll)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.messages.isEmpty) {
      return Center(
        child: Text(
          'No messages yet. Start the encrypted conversation.',
          style: Theme.of(context).textTheme.bodyMedium,
        ),
      );
    }

    return CustomScrollView(
      controller: _scrollController,
      reverse: true,
      physics: const BouncingScrollPhysics(),
      slivers: [
        if (widget.isLoadingOlder)
          const SliverToBoxAdapter(
            child: Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(child: CircularProgressIndicator()),
            ),
          ),
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 16),
          sliver: SliverList(
            delegate: SliverChildBuilderDelegate(
              (context, index) {
                final reversedIndex = widget.messages.length - 1 - index;
                final message = widget.messages[reversedIndex];
                final olderMessage = reversedIndex > 0
                    ? widget.messages[reversedIndex - 1]
                    : null;
                final newerMessage = reversedIndex < widget.messages.length - 1
                    ? widget.messages[reversedIndex + 1]
                    : null;

                final showDateHeader = olderMessage == null ||
                    !_isSameDay(message.createdAt, olderMessage.createdAt);
                final showAvatar = newerMessage == null ||
                    newerMessage.senderId != message.senderId ||
                    !_isSameMinute(newerMessage.createdAt, message.createdAt);
                final showSenderName = olderMessage == null ||
                    olderMessage.senderId != message.senderId;

                return Column(
                  children: [
                    if (showDateHeader)
                      _DateDivider(date: message.createdAt),
                    MessageBubbleWidget(
                      message: message,
                      showAvatar: showAvatar,
                      showSenderName: showSenderName,
                      onReply: widget.onReply == null
                          ? null
                          : () => widget.onReply!(message),
                      onCopy: widget.onCopy == null
                          ? null
                          : () => widget.onCopy!(message),
                      onEdit: widget.onEdit == null
                          ? null
                          : () => widget.onEdit!(message),
                      onDelete: widget.onDelete == null
                          ? null
                          : () => widget.onDelete!(message),
                      onToggleReaction: widget.onToggleReaction == null
                          ? null
                          : (emoji) => widget.onToggleReaction!(message, emoji),
                    ),
                  ],
                );
              },
              childCount: widget.messages.length,
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _handleScroll() async {
    if (widget.onLoadOlder == null || widget.isLoadingOlder) {
      return;
    }

    if (_scrollController.position.extentAfter < 240 && !_requestedOlder) {
      _requestedOlder = true;
      await widget.onLoadOlder!.call();
      if (mounted) {
        _requestedOlder = false;
      }
    }
  }

  bool _isSameDay(DateTime a, DateTime b) {
    return a.year == b.year && a.month == b.month && a.day == b.day;
  }

  bool _isSameMinute(DateTime a, DateTime b) {
    return _isSameDay(a, b) && a.hour == b.hour && a.minute == b.minute;
  }
}

class _DateDivider extends StatelessWidget {
  const _DateDivider({required this.date});

  final DateTime date;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final target = DateTime(date.year, date.month, date.day);
    final difference = today.difference(target).inDays;

    String label;
    if (difference == 0) {
      label = 'Today';
    } else if (difference == 1) {
      label = 'Yesterday';
    } else {
      label = '${date.year}.${date.month.toString().padLeft(2, '0')}.${date.day.toString().padLeft(2, '0')}';
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainer,
            borderRadius: BorderRadius.circular(999),
          ),
          child: Text(
            label,
            style: theme.textTheme.labelMedium,
          ),
        ),
      ),
    );
  }
}
