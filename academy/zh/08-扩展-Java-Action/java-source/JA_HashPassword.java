// ============================================================
// JA_HashPassword — BCrypt 密码哈希 Java Action
// 在 Mendix Studio Pro 中创建同名 Java Action 后，粘贴此内容
// 依赖：bcrypt-0.10.2.jar（放入项目 userlib/ 目录）
// ============================================================
package helpdesk.actions;

import com.mendix.systemwideinterfaces.core.IContext;
import com.mendix.webui.CustomJavaAction;
import at.favre.lib.crypto.bcrypt.BCrypt;

public class JA_HashPassword extends CustomJavaAction<String> {

    private final String password;

    public JA_HashPassword(IContext context, String password) {
        super(context);
        this.password = password;
    }

    @Override
    public String executeAction() throws Exception {
        if (password == null || password.isEmpty()) {
            throw new RuntimeException("Password cannot be empty");
        }
        // cost factor 12 — good balance between security and performance
        return BCrypt.withDefaults().hashToString(12, password.toCharArray());
    }
}
