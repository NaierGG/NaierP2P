import { Shield, UserCircle2, Users } from "lucide-react";

import type { ChannelMember } from "@/shared/types";

interface MemberBarProps {
  members: ChannelMember[];
}

export default function MemberBar({ members }: MemberBarProps) {
  if (members.length === 0) {
    return null;
  }

  return (
    <div className="border-b border-border/70 bg-card/35 px-6 py-4">
      <div className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.22em] text-primary/75">
        <Users className="h-3.5 w-3.5" />
        <span>Members</span>
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        {members.map((member) => (
          <div
            key={member.user_id}
            className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-secondary/55 px-3 py-1.5 text-sm shadow-[inset_0_1px_0_rgba(255,255,255,0.02)]"
          >
            <UserCircle2 className="h-4 w-4 text-primary" />
            <span className="max-w-[180px] truncate">
              {member.display_name || member.username}
            </span>
            {member.role !== "member" && (
              <span className="inline-flex items-center gap-1 rounded-full border border-primary/20 bg-primary/10 px-2 py-0.5 text-[11px] text-primary">
                <Shield className="h-3 w-3" />
                {member.role}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
