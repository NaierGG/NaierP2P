import 'package:flutter/material.dart';

import '../../shared/models/chat_message.dart';

class MessageBubbleWidget extends StatelessWidget {
  const MessageBubbleWidget({
    super.key,
    required this.message,
    required this.showAvatar,
    required this.showSenderName,
    this.onReply,
    this.onCopy,
    this.onEdit,
    this.onDelete,
    this.onToggleReaction,
  });

  final ChatMessage message;
  final bool showAvatar;
  final bool showSenderName;
  final VoidCallback? onReply;
  final VoidCallback? onCopy;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;
  final ValueChanged<String>? onToggleReaction;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final alignment =
        message.isOwn ? Alignment.centerRight : Alignment.centerLeft;
    final bubbleColor = message.isOwn
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerHighest;

    return Align(
      alignment: alignment,
      child: Row(
        mainAxisAlignment:
            message.isOwn ? MainAxisAlignment.end : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          if (!message.isOwn)
            SizedBox(
              width: 32,
              child: showAvatar
                  ? CircleAvatar(
                      radius: 14,
                      backgroundColor: theme.colorScheme.secondaryContainer,
                      child: Text(
                        message.senderName.isEmpty
                            ? '?'
                            : message.senderName[0].toUpperCase(),
                        style: theme.textTheme.labelMedium,
                      ),
                    )
                  : const SizedBox.shrink(),
            ),
          Flexible(
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: InkWell(
                borderRadius: BorderRadius.circular(24),
                onLongPress: () => _showActions(context),
                child: Column(
                  crossAxisAlignment: message.isOwn
                      ? CrossAxisAlignment.end
                      : CrossAxisAlignment.start,
                  children: [
                    if (showSenderName && !message.isOwn)
                      Padding(
                        padding: const EdgeInsets.only(left: 12, bottom: 4),
                        child: Text(
                          message.senderName,
                          style: theme.textTheme.labelMedium?.copyWith(
                            color: theme.colorScheme.primary,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                    Container(
                      constraints: const BoxConstraints(maxWidth: 320),
                      padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
                      decoration: BoxDecoration(
                        color: bubbleColor,
                        borderRadius: BorderRadius.circular(24),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (message.replyPreview != null)
                            Container(
                              width: double.infinity,
                              margin: const EdgeInsets.only(bottom: 8),
                              padding: const EdgeInsets.all(10),
                              decoration: BoxDecoration(
                                color:
                                    theme.colorScheme.surface.withOpacity(0.55),
                                borderRadius: BorderRadius.circular(16),
                              ),
                              child: Text(
                                message.replyPreview!,
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                                style: theme.textTheme.bodySmall,
                              ),
                            ),
                          _MessageBody(message: message),
                          const SizedBox(height: 6),
                          Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(
                                _formatTime(message.createdAt),
                                style: theme.textTheme.labelSmall,
                              ),
                              if (message.isEdited) ...[
                                const SizedBox(width: 6),
                                Text(
                                  'edited',
                                  style: theme.textTheme.labelSmall,
                                ),
                              ],
                              if (message.isOwn) ...[
                                const SizedBox(width: 6),
                                Icon(
                                  _statusIcon(message.status),
                                  size: 14,
                                  color: _statusColor(theme, message.status),
                                ),
                              ],
                            ],
                          ),
                        ],
                      ),
                    ),
                    if (message.reactions.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 6, left: 6),
                        child: Wrap(
                          spacing: 6,
                          runSpacing: 6,
                          children: message.reactions
                              .map(
                                (reaction) => InkWell(
                                  onTap: onToggleReaction == null
                                      ? null
                                      : () => onToggleReaction!(reaction.emoji),
                                  borderRadius: BorderRadius.circular(999),
                                  child: Container(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 10,
                                      vertical: 5,
                                    ),
                                    decoration: BoxDecoration(
                                      color: reaction.isMine
                                          ? theme
                                              .colorScheme
                                              .secondaryContainer
                                          : theme.colorScheme.surfaceContainer,
                                      borderRadius: BorderRadius.circular(999),
                                    ),
                                    child: Text(
                                      '${reaction.emoji} ${reaction.count}',
                                      style: theme.textTheme.labelMedium,
                                    ),
                                  ),
                                ),
                              )
                              .toList(),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _showActions(BuildContext context) async {
    final action = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.reply_outlined),
              title: const Text('Reply'),
              onTap: () => Navigator.of(context).pop('reply'),
            ),
            ListTile(
              leading: const Icon(Icons.copy_all_outlined),
              title: const Text('Copy'),
              onTap: () => Navigator.of(context).pop('copy'),
            ),
            if (message.isOwn)
              ListTile(
                leading: const Icon(Icons.edit_outlined),
                title: const Text('Edit'),
                onTap: () => Navigator.of(context).pop('edit'),
              ),
            if (message.isOwn)
              ListTile(
                leading: const Icon(Icons.delete_outline),
                title: const Text('Delete'),
                onTap: () => Navigator.of(context).pop('delete'),
              ),
          ],
        ),
      ),
    );

    if (action == 'reply') {
      onReply?.call();
    } else if (action == 'copy') {
      onCopy?.call();
    } else if (action == 'edit') {
      onEdit?.call();
    } else if (action == 'delete') {
      onDelete?.call();
    }
  }

  static String _formatTime(DateTime value) {
    final hour = value.hour.toString().padLeft(2, '0');
    final minute = value.minute.toString().padLeft(2, '0');
    return '$hour:$minute';
  }

  static IconData _statusIcon(ChatMessageDeliveryStatus status) {
    switch (status) {
      case ChatMessageDeliveryStatus.sending:
        return Icons.schedule_rounded;
      case ChatMessageDeliveryStatus.sent:
        return Icons.done_rounded;
      case ChatMessageDeliveryStatus.failed:
        return Icons.error_outline_rounded;
      case ChatMessageDeliveryStatus.read:
        return Icons.done_all_rounded;
    }
  }

  static Color _statusColor(ThemeData theme, ChatMessageDeliveryStatus status) {
    switch (status) {
      case ChatMessageDeliveryStatus.failed:
        return theme.colorScheme.error;
      case ChatMessageDeliveryStatus.read:
        return theme.colorScheme.primary;
      case ChatMessageDeliveryStatus.sending:
      case ChatMessageDeliveryStatus.sent:
        return theme.colorScheme.onSurfaceVariant;
    }
  }
}

class _MessageBody extends StatelessWidget {
  const _MessageBody({required this.message});

  final ChatMessage message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    switch (message.type) {
      case ChatMessageType.image:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              height: 180,
              width: 220,
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(18),
                gradient: LinearGradient(
                  colors: [
                    theme.colorScheme.secondaryContainer,
                    theme.colorScheme.primaryContainer,
                  ],
                ),
              ),
              alignment: Alignment.center,
              child: const Icon(Icons.image_outlined, size: 42),
            ),
            if (message.content.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(message.content),
            ],
          ],
        );
      case ChatMessageType.file:
        return Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface.withOpacity(0.5),
            borderRadius: BorderRadius.circular(18),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.attach_file_rounded),
              const SizedBox(width: 8),
              Flexible(
                child: Text(
                  message.attachmentLabel ?? message.content,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        );
      case ChatMessageType.text:
        return SelectableText(
          message.content,
          style: theme.textTheme.bodyMedium,
        );
    }
  }
}
