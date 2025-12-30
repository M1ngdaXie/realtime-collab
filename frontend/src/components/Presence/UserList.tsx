import { useEffect, useState } from "react";

interface UserListProps {
  awareness: any;
}

interface User {
  clientId: number;
  name: string;
  color: string;
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
    <div className="user-list">
      <h4>Online Users ({users.length})</h4>
      <div className="users">
        {users.map((user) => (
          <div key={user.clientId} className="user">
            <span
              className="user-color"
              style={{ backgroundColor: user.color }}
            ></span>
            <span className="user-name">{user.name}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default UserList;
