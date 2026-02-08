import { useEffect, useState } from "react";
import { Awareness } from "y-protocols/awareness";

interface UserListProps {
  awareness: Awareness;
}

interface User {
  clientId: number;
  name: string;
  color: string;
  avatar?: string;
}

const UserList = ({ awareness }: UserListProps) => {
  const [users, setUsers] = useState<User[]>([]);

  useEffect(() => {
    const updateUsers = () => {
      const states = awareness.getStates();
      const userList: User[] = [];

      states.forEach((state: any, clientId: number) => {
        if (state.user) {
          userList.push({
            clientId,
            name: state.user.name,
            color: state.user.color,
            avatar: state.user.avatar,
          });
        }
      });

      setUsers(userList);
    };

    updateUsers();
    awareness.on("change", updateUsers);

    return () => {
      awareness.off("change", updateUsers);
    };
  }, [awareness]);

  return (
    <div className="p-2">
      {/* Header with online count */}
      <div className="
        flex
        items-center
        gap-2
        mb-3
        pb-2
        border-b
        border-border
      ">
        <div className="
          w-2.5
          h-2.5
          bg-brand-teal
          animate-pulse
          rounded-full
        " />
        <h4 className="font-display text-xs text-text-secondary uppercase tracking-wide">
          Online
        </h4>
        <span className="
          ml-auto
          font-sans
          text-sm
          font-semibold
          text-text-primary
          bg-surface-muted
          px-2
          py-0.5
          rounded-full
        ">
          {users.length}
        </span>
      </div>

      {/* User list */}
      <div className="flex flex-col gap-2">
        {users.length === 0 ? (
          <div className="
            text-center
            py-4
            font-sans
            text-xs
            text-text-muted
          ">
            No users online
          </div>
        ) : (
          users.map((user) => (
            <div
              key={user.clientId}
              className="
                group
                flex
                items-center
                gap-3
                py-2
              "
            >
              {/* Avatar with online indicator */}
              <div className="relative flex-shrink-0">
                {user.avatar ? (
                  <>
                    <img
                      src={user.avatar}
                      alt={user.name}
                      className="
                        w-10
                        h-10
                        rounded-full
                        border
                        border-border
                        shadow-soft
                        object-cover
                      "
                      onError={(e) => {
                        // Fallback to colored circle on image error
                        e.currentTarget.style.display = 'none';
                        const fallback = e.currentTarget.nextElementSibling as HTMLElement;
                        if (fallback) {
                          fallback.style.display = 'flex';
                        }
                      }}
                    />
                    {/* Fallback colored square (hidden if avatar loads) */}
                    <div
                      className="w-10 h-10 border border-border shadow-soft items-center justify-center font-display text-xs text-text-secondary rounded-full bg-surface-muted"
                      style={{ display: 'none' }}
                    >
                      {user.name.charAt(0).toUpperCase()}
                    </div>
                  </>
                ) : (
                  <div
                    className="w-10 h-10 border border-border shadow-soft flex items-center justify-center font-display text-xs text-text-secondary rounded-full bg-surface-muted"
                  >
                    {user.name.charAt(0).toUpperCase()}
                  </div>
                )}

                {/* Online indicator dot */}
                <div className="
                  absolute
                  -bottom-0.5
                  -right-0.5
                  w-3
                  h-3
                  bg-brand-teal
                  border-2
                  border-white
                  shadow-soft
                  rounded-full
                " />
              </div>

              {/* User name */}
              <span className="
                font-sans
                text-sm
                font-medium
                text-text-primary
                truncate
                flex-1
              ">
                {user.name}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default UserList;
