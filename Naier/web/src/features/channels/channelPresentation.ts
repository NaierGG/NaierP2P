import type { Channel, ChannelMember } from "@/shared/types";

export interface ChannelPresentation {
  title: string;
  preview: string;
  meta: string;
}

export function getChannelPresentation(
  channel: Channel,
  members: ChannelMember[] | undefined,
  currentUserId: string | null,
  showPreview: boolean
): ChannelPresentation {
  const title = getChannelTitle(channel, members, currentUserId);
  const previewSource =
    channel.last_message?.content ||
    (channel.type === "dm" ? getParticipantSummary(members, currentUserId) : "") ||
    channel.description ||
    channel.type;

  return {
    title,
    preview: showPreview ? previewSource : channel.description || title,
    meta: getChannelMeta(channel, members, currentUserId),
  };
}

function getChannelTitle(
  channel: Channel,
  members: ChannelMember[] | undefined,
  currentUserId: string | null
): string {
  if (channel.type !== "dm") {
    return channel.name || "Untitled channel";
  }

  const peers = (members ?? []).filter((member) => member.user_id !== currentUserId);
  if (peers.length === 0) {
    return channel.name || "Direct message";
  }

  const [firstPeer, ...rest] = peers;
  const firstName = formatMemberName(firstPeer);
  if (rest.length === 0) {
    return firstName;
  }

  return `${firstName} +${rest.length}`;
}

function getChannelMeta(
  channel: Channel,
  members: ChannelMember[] | undefined,
  currentUserId: string | null
): string {
  if (channel.type === "dm") {
    const peerSummary = getParticipantSummary(members, currentUserId);
    return peerSummary || "Direct message";
  }

  const typeLabel =
    channel.type === "group" ? "Group" : channel.type === "public" ? "Public" : "Direct";
  return `${channel.member_count} members · ${typeLabel}`;
}

function getParticipantSummary(
  members: ChannelMember[] | undefined,
  currentUserId: string | null
): string {
  const peers = (members ?? []).filter((member) => member.user_id !== currentUserId);
  if (peers.length === 0) {
    return "";
  }

  return peers.map(formatMemberName).join(", ");
}

function formatMemberName(member: ChannelMember): string {
  return member.display_name || member.username || "Unknown user";
}
