import 'dart:async';

import 'package:flutter/material.dart';

import '../../shared/models/chat_message.dart';

enum MessageAttachmentKind {
  camera,
  gallery,
  file,
}

class ComposerAttachment {
  const ComposerAttachment({
    required this.kind,
    required this.label,
  });

  final MessageAttachmentKind kind;
  final String label;
}

class MessageInputResult {
  const MessageInputResult({
    required this.text,
    required this.clientEventId,
    this.replyTo,
    this.editingMessage,
    this.attachment,
  });

  final String text;
  final String clientEventId;
  final ChatMessage? replyTo;
  final ChatMessage? editingMessage;
  final ComposerAttachment? attachment;
}

class MessageInputWidget extends StatefulWidget {
  const MessageInputWidget({
    super.key,
    this.replyTo,
    this.editingMessage,
    this.onCancelReply,
    this.onCancelEdit,
    this.onSend,
    this.onTypingChanged,
  });

  final ChatMessage? replyTo;
  final ChatMessage? editingMessage;
  final VoidCallback? onCancelReply;
  final VoidCallback? onCancelEdit;
  final ValueChanged<MessageInputResult>? onSend;
  final ValueChanged<bool>? onTypingChanged;

  @override
  State<MessageInputWidget> createState() => _MessageInputWidgetState();
}

class _MessageInputWidgetState extends State<MessageInputWidget> {
  final TextEditingController _controller = TextEditingController();
  final FocusNode _focusNode = FocusNode();
  Timer? _typingDebounce;
  ComposerAttachment? _attachment;
  String? _editingId;

  @override
  void initState() {
    super.initState();
    _primeEditingState();
  }

  @override
  void didUpdateWidget(covariant MessageInputWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.editingMessage?.id != widget.editingMessage?.id) {
      _primeEditingState();
    }
  }

  @override
  void dispose() {
    _typingDebounce?.cancel();
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface.withValues(alpha: 0.96),
          border: Border(
            top: BorderSide(
              color: theme.colorScheme.outlineVariant,
            ),
          ),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (widget.replyTo != null || widget.editingMessage != null)
              _ContextBanner(
                replyTo: widget.replyTo,
                editingMessage: widget.editingMessage,
                onClose: widget.editingMessage != null
                    ? widget.onCancelEdit
                    : widget.onCancelReply,
              ),
            if (_attachment != null)
              _AttachmentChip(
                attachment: _attachment!,
                onRemove: () => setState(() => _attachment = null),
              ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                IconButton(
                  onPressed: _pickAttachment,
                  icon: const Icon(Icons.add_circle_outline_rounded),
                ),
                Expanded(
                  child: TextField(
                    controller: _controller,
                    focusNode: _focusNode,
                    minLines: 1,
                    maxLines: 6,
                    textInputAction: TextInputAction.newline,
                    onChanged: _handleChanged,
                    decoration: InputDecoration(
                      hintText: widget.editingMessage != null
                          ? 'Edit encrypted message'
                          : 'Write an encrypted message',
                      filled: true,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(28),
                        borderSide: BorderSide.none,
                      ),
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 18,
                        vertical: 14,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  onPressed: () {},
                  icon: const Icon(Icons.sentiment_satisfied_alt_outlined),
                ),
                FilledButton(
                  onPressed: _submit,
                  style: FilledButton.styleFrom(
                    minimumSize: const Size(54, 54),
                    shape: const CircleBorder(),
                    padding: EdgeInsets.zero,
                  ),
                  child: Icon(
                    widget.editingMessage != null
                        ? Icons.check_rounded
                        : Icons.arrow_upward_rounded,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  void _primeEditingState() {
    if (widget.editingMessage != null &&
        widget.editingMessage!.id != _editingId) {
      _editingId = widget.editingMessage!.id;
      _controller
        ..text = widget.editingMessage!.content
        ..selection = TextSelection.collapsed(
          offset: widget.editingMessage!.content.length,
        );
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _focusNode.requestFocus();
        }
      });
      return;
    }

    if (widget.editingMessage == null && _editingId != null) {
      _editingId = null;
      _controller.clear();
    }
  }

  Future<void> _pickAttachment() async {
    final kind = await showModalBottomSheet<MessageAttachmentKind>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_camera_back_outlined),
              title: const Text('Camera'),
              onTap: () => Navigator.of(context).pop(MessageAttachmentKind.camera),
            ),
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: const Text('Gallery'),
              onTap: () => Navigator.of(context).pop(MessageAttachmentKind.gallery),
            ),
            ListTile(
              leading: const Icon(Icons.attach_file_rounded),
              title: const Text('File'),
              onTap: () => Navigator.of(context).pop(MessageAttachmentKind.file),
            ),
          ],
        ),
      ),
    );

    if (kind == null || !mounted) {
      return;
    }

    setState(() {
      _attachment = ComposerAttachment(
        kind: kind,
        label: switch (kind) {
          MessageAttachmentKind.camera => 'camera_capture.jpg',
          MessageAttachmentKind.gallery => 'gallery_image.webp',
          MessageAttachmentKind.file => 'secure_document.pdf',
        },
      );
    });
  }

  void _handleChanged(String value) {
    widget.onTypingChanged?.call(value.trim().isNotEmpty);
    _typingDebounce?.cancel();
    _typingDebounce = Timer(const Duration(seconds: 2), () {
      widget.onTypingChanged?.call(false);
    });
  }

  void _submit() {
    final text = _controller.text.trim();
    if (text.isEmpty && _attachment == null) {
      return;
    }

    widget.onSend?.call(
      MessageInputResult(
        text: text,
        clientEventId: 'mobile-${DateTime.now().microsecondsSinceEpoch}',
        replyTo: widget.replyTo,
        editingMessage: widget.editingMessage,
        attachment: _attachment,
      ),
    );

    _typingDebounce?.cancel();
    widget.onTypingChanged?.call(false);
    _controller.clear();
    setState(() => _attachment = null);
    if (widget.editingMessage != null) {
      widget.onCancelEdit?.call();
    } else if (widget.replyTo != null) {
      widget.onCancelReply?.call();
    }
  }
}

class _ContextBanner extends StatelessWidget {
  const _ContextBanner({
    required this.replyTo,
    required this.editingMessage,
    required this.onClose,
  });

  final ChatMessage? replyTo;
  final ChatMessage? editingMessage;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    final target = editingMessage ?? replyTo;
    if (target == null) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(18),
      ),
      child: Row(
        children: [
          Container(
            width: 4,
            height: 40,
            decoration: BoxDecoration(
              color: theme.colorScheme.primary,
              borderRadius: BorderRadius.circular(999),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  editingMessage != null ? 'Editing message' : 'Replying to ${target.senderName}',
                  style: theme.textTheme.labelLarge,
                ),
                const SizedBox(height: 2),
                Text(
                  target.content,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: onClose,
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
    );
  }
}

class _AttachmentChip extends StatelessWidget {
  const _AttachmentChip({
    required this.attachment,
    required this.onRemove,
  });

  final ComposerAttachment attachment;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Chip(
          avatar: Icon(
            switch (attachment.kind) {
              MessageAttachmentKind.camera => Icons.photo_camera_back_outlined,
              MessageAttachmentKind.gallery => Icons.photo_library_outlined,
              MessageAttachmentKind.file => Icons.attach_file_rounded,
            },
          ),
          label: Text(attachment.label),
          onDeleted: onRemove,
        ),
      ),
    );
  }
}
