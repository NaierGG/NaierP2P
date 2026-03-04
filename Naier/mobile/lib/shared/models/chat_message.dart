enum ChatMessageType {
  text,
  image,
  file,
}

enum ChatMessageDeliveryStatus {
  sending,
  sent,
  failed,
  read,
}

class ChatMessageReaction {
  const ChatMessageReaction({
    required this.emoji,
    required this.count,
    this.isMine = false,
  });

  final String emoji;
  final int count;
  final bool isMine;
}

class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.channelId,
    required this.senderId,
    required this.senderName,
    required this.content,
    required this.createdAt,
    required this.type,
    this.replyPreview,
    this.attachmentLabel,
    this.clientEventId,
    this.serverEventId,
    this.sequence,
    this.isOwn = false,
    this.isEdited = false,
    this.status = ChatMessageDeliveryStatus.sent,
    this.reactions = const <ChatMessageReaction>[],
  });

  final String id;
  final String channelId;
  final String senderId;
  final String senderName;
  final String content;
  final DateTime createdAt;
  final ChatMessageType type;
  final String? replyPreview;
  final String? attachmentLabel;
  final String? clientEventId;
  final String? serverEventId;
  final int? sequence;
  final bool isOwn;
  final bool isEdited;
  final ChatMessageDeliveryStatus status;
  final List<ChatMessageReaction> reactions;

  ChatMessage copyWith({
    String? id,
    String? channelId,
    String? senderId,
    String? senderName,
    String? content,
    DateTime? createdAt,
    ChatMessageType? type,
    String? replyPreview,
    String? attachmentLabel,
    String? clientEventId,
    String? serverEventId,
    int? sequence,
    bool? isOwn,
    bool? isEdited,
    ChatMessageDeliveryStatus? status,
    List<ChatMessageReaction>? reactions,
  }) {
    return ChatMessage(
      id: id ?? this.id,
      channelId: channelId ?? this.channelId,
      senderId: senderId ?? this.senderId,
      senderName: senderName ?? this.senderName,
      content: content ?? this.content,
      createdAt: createdAt ?? this.createdAt,
      type: type ?? this.type,
      replyPreview: replyPreview ?? this.replyPreview,
      attachmentLabel: attachmentLabel ?? this.attachmentLabel,
      clientEventId: clientEventId ?? this.clientEventId,
      serverEventId: serverEventId ?? this.serverEventId,
      sequence: sequence ?? this.sequence,
      isOwn: isOwn ?? this.isOwn,
      isEdited: isEdited ?? this.isEdited,
      status: status ?? this.status,
      reactions: reactions ?? this.reactions,
    );
  }
}
