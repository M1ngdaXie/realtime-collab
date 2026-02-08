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
    <div className="
      bg-pixel-white
      border-[3px]
      border-pixel-outline
      shadow-pixel-md
      p-4
    ">
      {/* Header with online count */}
      <div className="
        flex
        items-center
        gap-2
        mb-4
        pb-3
        border-b-[2px]
        border-pixel-outline
      ">
        <div className="
          w-3
          h-3
          bg-pixel-green-lime
          animate-pulse
          border-[2px]
          border-pixel-outline
        " />
        <h4 className="font-pixel text-xs text-pixel-text-primary">
          ONLINE
        </h4>
        <span className="
          ml-auto
          font-sans
          text-sm
          font-bold
          text-pixel-purple-bright
          bg-pixel-panel
          px-2
          py-1
          border-[2px]
          border-pixel-outline
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
            text-pixel-text-muted
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
                p-2
                bg-pixel-panel
                border-[2px]
                border-pixel-outline
                shadow-pixel-sm
                hover:shadow-pixel-hover
                hover:-translate-y-0.5
                hover:-translate-x-0.5
                transition-all
                duration-75
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
                        border-[3px]
                        border-pixel-outline
                        shadow-pixel-sm
                        object-cover
                      "
                      onError={(e) => {
                        // Fallback to colored square on image error
                        e.currentTarget.style.display = 'none';
                        const fallback = e.currentTarget.nextElementSibling as HTMLElement;
                        if (fallback) {
                          fallback.style.display = 'flex';
                        }
                      }}
                    />
                    {/* Fallback colored square (hidden if avatar loads) */}
                    <div
                      className="w-10 h-10 border-[3px] border-pixel-outline shadow-pixel-sm items-center justify-center font-pixel text-xs text-white"
                      style={{ backgroundColor: user.color, display: 'none' }}
                    >
                      {user.name.charAt(0).toUpperCase()}
                    </div>
                  </>
                ) : (
                  <div
                    className="w-10 h-10 border-[3px] border-pixel-outline shadow-pixel-sm flex items-center justify-center font-pixel text-xs text-white"
                    style={{ backgroundColor: user.color }}
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
                  bg-pixel-green-lime
                  border-[2px]
                  border-pixel-white
                  shadow-pixel-sm
                " />
              </div>

              {/* User name */}
              <span className="
                font-sans
                text-sm
                font-medium
                text-pixel-text-primary
                truncate
                flex-1
              ">
                {user.name}
              </span>

              {/* User color indicator (small square) */}
              <div
                className="
                  flex-shrink-0
                  w-4
                  h-4
                  border-[2px]
                  border-pixel-outline
                  shadow-pixel-sm
                "
                style={{ backgroundColor: user.color }}
              />
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default UserList;
